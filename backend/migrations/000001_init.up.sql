CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE imports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename        TEXT NOT NULL,
    file_hash       TEXT NOT NULL,
    storage_path    TEXT NOT NULL,
    file_type       TEXT NOT NULL CHECK (file_type IN ('xlsx', 'csv')),
    validated_month DATE,
    status          TEXT NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    transaction_count INTEGER NOT NULL DEFAULT 0,
    error_code      TEXT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT imports_file_hash_unique UNIQUE (file_hash)
);

CREATE INDEX idx_imports_status ON imports (status);
CREATE INDEX idx_imports_validated_month ON imports (validated_month DESC);

CREATE TABLE transactions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    import_id               UUID NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    transaction_date        DATE NOT NULL,
    original_description    TEXT NOT NULL,
    merchant                TEXT,
    amount_paise            BIGINT NOT NULL CHECK (amount_paise >= 0),
    currency                TEXT NOT NULL DEFAULT 'INR',
    direction               TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
    transaction_type        TEXT,
    category                TEXT,
    subcategory             TEXT,
    classification_status   TEXT NOT NULL DEFAULT 'pending'
                                CHECK (classification_status IN (
                                    'pending', 'classified', 'needs_review', 'corrected'
                                )),
    dedup_hash              TEXT NOT NULL,
    raw_data                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_dedup_unique UNIQUE (dedup_hash)
);

CREATE INDEX idx_transactions_import_id ON transactions (import_id);
CREATE INDEX idx_transactions_date ON transactions (transaction_date);
CREATE INDEX idx_transactions_classification_status ON transactions (classification_status);
CREATE INDEX idx_transactions_category ON transactions (category);

CREATE TABLE classifications (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id          UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    model_name              TEXT NOT NULL,
    model_version           TEXT,
    predicted_category      TEXT,
    predicted_subcategory   TEXT,
    predicted_type          TEXT,
    confidence              NUMERIC(5,4) NOT NULL DEFAULT 0,
    reasoning               TEXT,
    needs_review            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_classifications_transaction_id ON classifications (transaction_id);
CREATE INDEX idx_classifications_needs_review ON classifications (needs_review);

CREATE TABLE user_feedback (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id          UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    previous_category       TEXT,
    previous_subcategory    TEXT,
    corrected_category      TEXT NOT NULL,
    corrected_subcategory   TEXT,
    corrected_type          TEXT,
    note                    TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_feedback_transaction_id ON user_feedback (transaction_id);

CREATE TABLE user_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern             TEXT NOT NULL,
    match_type          TEXT NOT NULL DEFAULT 'contains'
                            CHECK (match_type IN ('contains', 'equals', 'regex')),
    category            TEXT NOT NULL,
    subcategory         TEXT,
    transaction_type    TEXT,
    priority            INTEGER NOT NULL DEFAULT 0,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_rules_active_priority ON user_rules (active, priority DESC);
