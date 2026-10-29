CREATE TABLE month_health_reports (
    month          DATE PRIMARY KEY,
    score          INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    verdict        TEXT NOT NULL,
    tip            TEXT NOT NULL DEFAULT '',
    signals        JSONB NOT NULL DEFAULT '{}',
    model_name     TEXT NOT NULL DEFAULT '',
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
