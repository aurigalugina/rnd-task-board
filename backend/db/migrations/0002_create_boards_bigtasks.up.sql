CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    tag TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE big_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    deadline DATE NOT NULL,
    default_pic_user_id UUID REFERENCES users(id),
    on_hold BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE big_task_signoffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    big_task_id UUID NOT NULL UNIQUE REFERENCES big_tasks(id) ON DELETE CASCADE,
    signed_by UUID NOT NULL REFERENCES users(id),
    signed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
