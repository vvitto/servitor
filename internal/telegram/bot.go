package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/vitalijnizankovskij/servitor/internal/ingest"
	"github.com/vitalijnizankovskij/servitor/internal/store"
)

type Service struct {
	bot      *bot.Bot
	db       *store.DB
	ingester *ingest.Ingester
	log      *slog.Logger
	username string
}

type Options struct {
	Token string
	DB    *store.DB
	Log   *slog.Logger
	Debug bool
}

func New(ctx context.Context, opts Options) (*Service, error) {
	s := &Service{
		db:       opts.DB,
		log:      opts.Log,
		ingester: ingest.New(opts.DB),
	}

	botOpts := []bot.Option{
		bot.WithDefaultHandler(s.handleUpdate),
		bot.WithAllowedUpdates(bot.AllowedUpdates{"message", "edited_message"}),
		bot.WithErrorsHandler(func(err error) {
			opts.Log.Error("bot error", "err", err)
		}),
	}
	if opts.Debug {
		botOpts = append(botOpts, bot.WithDebug())
	}

	b, err := bot.New(opts.Token, botOpts...)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}
	s.bot = b

	me, err := b.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("getMe: %w", err)
	}
	s.username = me.Username

	s.log.Info("bot ready", "username", me.Username, "id", me.ID)
	return s, nil
}

func (s *Service) Start(ctx context.Context) { s.bot.Start(ctx) }

func (s *Service) handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	payload, err := json.Marshal(update)
	if err != nil {
		s.log.Error("marshal update", "err", err, "update_id", update.ID)
		return
	}

	res, err := s.ingester.Handle(ctx, update, payload)
	if err != nil {
		s.log.Error("ingest", "err", err, "update_id", update.ID)
		return
	}
	if res.Duplicate {
		s.log.Debug("duplicate update ignored", "update_id", update.ID)
		return
	}
	if res.Message != nil {
		s.log.Debug("stored message",
			"chat_id", res.Message.ChatID,
			"message_id", res.Message.MessageID,
			"kind", res.Message.Kind)
	}
}

func (s *Service) record(ctx context.Context, update *models.Update) {
	payload, err := json.Marshal(update)
	if err != nil {
		return
	}
	if _, err := s.ingester.Handle(ctx, update, payload); err != nil {
		s.log.Error("ingest command", "err", err, "update_id", update.ID)
	}
}

func (s *Service) reply(ctx context.Context, msg *models.Message, text string) {
	_, err := s.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID:                msg.ID,
			AllowSendingWithoutReply: true,
		},
	})
	if err != nil {
		s.log.Error("send", "err", err, "chat_id", msg.Chat.ID)
	}
}
