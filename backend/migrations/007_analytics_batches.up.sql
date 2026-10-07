-- The ledger makes retries safe after a successful commit but failed Redis ACK.
CREATE TABLE analytics_batches (
    id TEXT PRIMARY KEY,
    persisted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
