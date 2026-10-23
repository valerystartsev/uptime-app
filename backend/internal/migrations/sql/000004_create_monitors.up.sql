CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    url TEXT NOT NULL CHECK (char_length(url) <= 2048),
    interval_value BIGINT NOT NULL CHECK (interval_value > 0),
    interval_unit TEXT NOT NULL CHECK (interval_unit IN ('seconds', 'minutes', 'hours')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX monitors_user_created_index ON monitors (user_id, created_at DESC);
