CREATE TABLE big_task_reviewers (
    big_task_id UUID NOT NULL REFERENCES big_tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (big_task_id, user_id)
);

CREATE INDEX idx_big_task_reviewers_user ON big_task_reviewers(user_id);
