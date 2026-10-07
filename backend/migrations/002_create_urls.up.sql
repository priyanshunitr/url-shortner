-- URL domain: urls table
CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    long_url TEXT NOT NULL,
    short_url TEXT UNIQUE NOT NULL,
    clicks BIGINT DEFAULT 0,
    expiry TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
