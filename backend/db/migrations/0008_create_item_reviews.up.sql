-- Lihat docs/decision-log/decision-log-review-queue-schema-20260809.md untuk
-- desain & alasan. Pola sama seperti big_task_signoffs: keberadaan baris =
-- status (di sini: sudah ditinjau).
CREATE TABLE item_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_type TEXT NOT NULL,
    item_id UUID NOT NULL,
    reviewed_by UUID NOT NULL REFERENCES users(id),
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_type, item_id)
);

CREATE INDEX idx_item_reviews_lookup ON item_reviews(item_type, item_id);
