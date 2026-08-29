package ingest

import (
	"context"
	"database/sql"

	"github.com/go-telegram/bot/models"

	"github.com/vitalijnizankovskij/servitor/internal/store"
)

type Ingester struct {
	db *store.DB
}

func New(db *store.DB) *Ingester {
	return &Ingester{db: db}
}

type Result struct {
	Duplicate bool
	Message   *store.Message
}

func (in *Ingester) Handle(ctx context.Context, u *models.Update, payload []byte) (Result, error) {
	var res Result

	msg := messageOf(u)
	var chatID *int64
	if msg != nil {
		id := msg.Chat.ID
		chatID = &id
	}

	tx, err := in.db.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}

	defer tx.Rollback()

	fresh, err := store.InsertUpdate(ctx, tx, u.ID, chatID, payload)
	if err != nil {
		return res, err
	}
	if !fresh {
		res.Duplicate = true
		return res, tx.Commit()
	}

	if msg == nil {
		return res, tx.Commit()
	}

	if err := in.storeMessage(ctx, tx, msg, &res); err != nil {
		return res, err
	}
	return res, tx.Commit()
}

func (in *Ingester) storeMessage(ctx context.Context, tx *sql.Tx, msg *models.Message, res *Result) error {
	if err := store.UpsertChat(ctx, tx, store.Chat{
		ID:       msg.Chat.ID,
		Type:     string(msg.Chat.Type),
		Title:    msg.Chat.Title,
		Username: msg.Chat.Username,
	}); err != nil {
		return err
	}

	// From is nil for channel posts, which have no individual author.
	if msg.From != nil {
		if err := store.UpsertUser(ctx, tx, store.User{
			ID:        msg.From.ID,
			IsBot:     msg.From.IsBot,
			FirstName: msg.From.FirstName,
			LastName:  msg.From.LastName,
			Username:  msg.From.Username,
		}, int64(msg.Date)); err != nil {
			return err
		}
	}

	stored := project(msg)
	if err := store.UpsertMessage(ctx, tx, stored); err != nil {
		return err
	}
	res.Message = &stored
	return nil
}

func project(msg *models.Message) store.Message {
	m := store.Message{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Date:      int64(msg.Date),
		Kind:      classify(msg),
		Text:      text(msg),
	}

	if msg.From != nil {
		id := msg.From.ID
		m.FromUserID = &id
	}
	if msg.EditDate != 0 {
		d := int64(msg.EditDate)
		m.EditDate = &d
	}
	if msg.ReplyToMessage != nil {
		r := msg.ReplyToMessage.ID
		m.ReplyToMessageID = &r
	}
	return m
}

func messageOf(u *models.Update) *models.Message {
	switch {
	case u.Message != nil:
		return u.Message
	case u.EditedMessage != nil:
		return u.EditedMessage
	case u.ChannelPost != nil:
		return u.ChannelPost
	case u.EditedChannelPost != nil:
		return u.EditedChannelPost
	default:
		return nil
	}
}

func text(msg *models.Message) string {
	if msg.Text != "" {
		return msg.Text
	}
	return msg.Caption
}

func classify(msg *models.Message) string {
	switch {
	case msg.Text != "":
		return "text"
	case len(msg.Photo) > 0:
		return "photo"
	case msg.Video != nil:
		return "video"
	case msg.Voice != nil:
		return "voice"
	case msg.Sticker != nil:
		return "sticker"
	case msg.Document != nil:
		return "document"
	case msg.Animation != nil:
		return "animation"
	case msg.Audio != nil:
		return "audio"
	default:
		return "other"
	}
}
