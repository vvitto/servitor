# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`servitor` is a Telegram bot that records the messages of the chats it is in into a local SQLite file. It long-polls the Bot API and has no HTTP server, no webhook, and (currently) no command surface — it only listens and writes.

## Commands

```bash
make build     # go build -o bin/servitor ./cmd/servitor
make lint      # gofmt -l . && go vet ./...
make run       # build, then ./bin/servitor
make clean     # rm -rf bin
go test ./...  # no test files exist yet; the Makefile declares `test` in .PHONY but has no target
```

Running requires env vars in the process (nothing in the code loads `.env` — copy `.env.example`, fill it, and `set -a; source .env; set +a` or export manually):

- `SERVITOR_BOT_TOKEN` — required, from @BotFather; startup fails without it.
- `SERVITOR_DB_PATH` — defaults to `servitor.db`, created on first run.
- `SERVITOR_DEBUG` — `true` turns on `slog` debug level *and* the go-telegram library's own request debug output.

## Architecture

Flow: `cmd/servitor` wires config → store → telegram service, then blocks on `svc.Start(ctx)` until SIGINT/SIGTERM cancels the context.

**The schema is deliberately two-layered** (see the package doc in `internal/store/db.go`):

- `updates` is an append-only log of the raw JSON payload of every Telegram update, keyed by `update_id`. It is the source of truth.
- `chats`, `users`, `messages` are *projections* — derived data that can be dropped and rebuilt from `updates`.

Consequence for any schema change: the projection tables can be reshaped freely (add a migration that drops and rebuilds them from `updates`), but never lose or lossily transform `updates.payload`.

**Ingest is one transaction per update** (`internal/ingest/ingest.go`). `store.InsertUpdate` uses `INSERT OR IGNORE` on the `update_id` primary key and returns whether the row was fresh; a non-fresh insert means Telegram redelivered the update, so the transaction commits and the projection writes are skipped. That is the entire dedupe mechanism — preserve it when adding new projections.

**Edits reuse the same row.** `messageOf` maps `edited_message` / `edited_channel_post` to the same `(chat_id, message_id)` key, and `UpsertMessage` overwrites `edit_date`, `kind`, `text` on conflict. Original text is not retained in `messages`; it is still in the `updates` log.

**`msg.From` is nil for channel posts.** Both `storeMessage` and `project` guard on this; any new field extraction must too.

### Migrations

`internal/store/migrations/*.sql` are embedded via `go:embed` and applied at `store.Open` in lexical filename order, each in its own transaction, recorded by filename in `schema_migrations`. Add a new numbered file (`0002_*.sql`); never edit an applied one. Migration files are executed as a single `ExecContext`, so multiple statements per file are fine.

`SetMaxOpenConns(1)` is intentional — SQLite via `modernc.org/sqlite` (pure Go, no cgo) with a single writer.

### Layering

`store` owns SQL and knows nothing of Telegram types. `ingest` is the only place that translates `models.Message` → `store.Message` (`project`, `classify`, `text`). `telegram` marshals the update and delegates; keep Telegram types out of `store`.

## Current state

`store.RecentMessages` and `store.Stats` are written but have no callers — the `/recent` and `/stats` style bot commands they were meant to back were removed from `internal/telegram/bot.go`. Reuse them if reintroducing a command surface rather than writing new queries.

`bot.WithAllowedUpdates` is limited to `message` and `edited_message`, so channel posts are not currently delivered even though `ingest` handles them.
