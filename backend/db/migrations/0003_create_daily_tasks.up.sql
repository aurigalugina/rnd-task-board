CREATE TABLE daily_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    big_task_id UUID NOT NULL REFERENCES big_tasks(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    pic_user_id UUID NOT NULL REFERENCES users(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE day_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_task_id UUID NOT NULL REFERENCES daily_tasks(id) ON DELETE CASCADE,
    entry_date DATE NOT NULL,
    planned_text TEXT NOT NULL DEFAULT '',
    is_done BOOLEAN NOT NULL DEFAULT false,
    blocker_text TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (daily_task_id, entry_date)
);
