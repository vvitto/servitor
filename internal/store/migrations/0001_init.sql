CREATE TABLE updates (
    update_id   INTEGER PRIMARY KEY,
    chat_id     INTEGER,
    received_at INTEGER NOT NULL,
    payload     TEXT    NOT NULL
);
CREATE INDEX idx_updates_chat ON updates(chat_id, received_at);

CREATE TABLE chats (
    id        INTEGER PRIMARY KEY,
    type      TEXT    NOT NULL,
    title     TEXT,
    username  TEXT,
    joined_at INTEGER NOT NULL
);

CREATE TABLE users (
    id         INTEGER PRIMARY KEY,
    is_bot     INTEGER NOT NULL DEFAULT 0,
    first_name TEXT,
    last_name  TEXT,
    username   TEXT,
    first_seen INTEGER NOT NULL,
    last_seen  INTEGER NOT NULL
);

CREATE TABLE messages (
    chat_id             INTEGER NOT NULL,
    message_id          INTEGER NOT NULL,
    from_user_id        INTEGER,
    date                INTEGER NOT NULL,
    edit_date           INTEGER,
    reply_to_message_id INTEGER,
    kind                TEXT    NOT NULL,
    text                TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (chat_id, message_id)
);
CREATE INDEX idx_messages_chat_date ON messages(chat_id, date DESC);
