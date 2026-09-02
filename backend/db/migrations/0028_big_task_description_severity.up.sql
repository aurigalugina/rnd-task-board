ALTER TABLE big_tasks ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE big_tasks ADD COLUMN severity TEXT NOT NULL DEFAULT 'medium'
    CHECK (severity IN ('critical', 'high', 'medium', 'low'));
