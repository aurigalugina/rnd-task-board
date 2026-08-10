CREATE TABLE weekly_push_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    big_task_id UUID NOT NULL REFERENCES big_tasks(id) ON DELETE CASCADE,
    week_start DATE NOT NULL,
    callback_id TEXT NOT NULL UNIQUE,
    pushed_by UUID NOT NULL REFERENCES users(id),
    pushed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_payload_actual_pct NUMERIC(5,2),
    last_payload_expected_pct NUMERIC(5,2),
    UNIQUE (big_task_id, week_start)
);
