CREATE TABLE year_health_reports (
    year           INTEGER PRIMARY KEY CHECK (year >= 1990 AND year <= 2100),
    score          INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    verdict        TEXT NOT NULL,
    tip            TEXT NOT NULL DEFAULT '',
    signals        JSONB NOT NULL DEFAULT '{}',
    model_name     TEXT NOT NULL DEFAULT '',
    generated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
