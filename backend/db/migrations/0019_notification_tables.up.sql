-- Global app config (key-value) — dipakai buat Telegram bot token dan setting
-- global lainnya. Value disimpan plaintext (internal tool, acceptable).
CREATE TABLE app_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-user notification settings
CREATE TABLE notification_settings (
    user_id                 UUID    PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    telegram_chat_id        TEXT,
    telegram_thread_id      TEXT,                -- opsional: buat group topic
    deadline_threshold_days INT     NOT NULL DEFAULT 3,
    cooldown_hours          INT     NOT NULL DEFAULT 24,
    notify_sign_off_ready   BOOLEAN NOT NULL DEFAULT true,
    notify_verdict_lose     BOOLEAN NOT NULL DEFAULT true,
    notify_deadline_soon    BOOLEAN NOT NULL DEFAULT true,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Alert log — dedup: jangan kirim alert yang sama ke user yang sama dalam
-- window cooldown_hours. ref_id = big_task_id yang di-alert.
CREATE TABLE notification_log (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_type TEXT        NOT NULL,
    ref_id     TEXT        NOT NULL,
    channel    TEXT        NOT NULL DEFAULT 'telegram',
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX notification_log_lookup
    ON notification_log (user_id, alert_type, ref_id, sent_at DESC);
