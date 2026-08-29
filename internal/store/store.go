package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Message struct {
	ChatID           int64
	MessageID        int
	FromUserID       *int64
	Date             int64
	EditDate         *int64
	ReplyToMessageID *int
	Kind             string
	Text             string
}

type Chat struct {
	ID       int64
	Type     string
	Title    string
	Username string
}

type User struct {
	ID        int64
	IsBot     bool
	FirstName string
	LastName  string
	Username  string
}

func InsertUpdate(ctx context.Context, tx *sql.Tx, updateID int64, chatID *int64, payload []byte) (bool, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO updates(update_id, chat_id, received_at, payload)
		 VALUES (?, ?, unixepoch(), ?)`,
		updateID, chatID, string(payload))
	if err != nil {
		return false, fmt.Errorf("insert update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func UpsertChat(ctx context.Context, tx *sql.Tx, c Chat) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO chats(id, type, title, username, joined_at)
		 VALUES (?, ?, ?, ?, unixepoch())
		 ON CONFLICT(id) DO UPDATE SET
		     type = excluded.type,
		     title = excluded.title,
		     username = excluded.username`,
		c.ID, c.Type, nullStr(c.Title), nullStr(c.Username))
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	return nil
}

func UpsertUser(ctx context.Context, tx *sql.Tx, u User, seenAt int64) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO users(id, is_bot, first_name, last_name, username, first_seen, last_seen)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     first_name = excluded.first_name,
		     last_name = excluded.last_name,
		     username = excluded.username,
		     last_seen = max(users.last_seen, excluded.last_seen)`,
		u.ID, u.IsBot, nullStr(u.FirstName), nullStr(u.LastName), nullStr(u.Username),
		seenAt, seenAt)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

func UpsertMessage(ctx context.Context, tx *sql.Tx, m Message) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO messages(chat_id, message_id, from_user_id, date, edit_date,
		     reply_to_message_id, kind, text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(chat_id, message_id) DO UPDATE SET
		     edit_date = excluded.edit_date,
		     kind = excluded.kind,
		     text = excluded.text`,
		m.ChatID, m.MessageID, m.FromUserID, m.Date, m.EditDate,
		m.ReplyToMessageID, m.Kind, m.Text)
	if err != nil {
		return fmt.Errorf("upsert message: %w", err)
	}
	return nil
}

func (db *DB) RecentMessages(ctx context.Context, chatID int64, limit int) ([]Message, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT chat_id, message_id, from_user_id, date, edit_date,
		        reply_to_message_id, kind, text
		 FROM messages WHERE chat_id = ?
		 ORDER BY date DESC, message_id DESC
		 LIMIT ?`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ChatID, &m.MessageID, &m.FromUserID, &m.Date,
			&m.EditDate, &m.ReplyToMessageID, &m.Kind, &m.Text); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

type Stats struct {
	Messages  int64
	Users     int64
	Updates   int64
	FirstSeen *int64 // nil when the chat has no messages yet
	LastSeen  *int64
}

func (db *DB) Stats(ctx context.Context, chatID int64) (Stats, error) {
	var s Stats
	err := db.QueryRowContext(ctx,
		`SELECT count(*), count(DISTINCT from_user_id), min(date), max(date)
		 FROM messages WHERE chat_id = ?`, chatID).
		Scan(&s.Messages, &s.Users, &s.FirstSeen, &s.LastSeen)
	if err != nil {
		return s, err
	}
	err = db.QueryRowContext(ctx,
		`SELECT count(*) FROM updates WHERE chat_id = ?`, chatID).Scan(&s.Updates)
	return s, err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
