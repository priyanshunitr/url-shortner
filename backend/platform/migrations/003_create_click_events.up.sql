-- Analytics domain: click_events table
CREATE TABLE IF NOT EXISTS click_events (
    id BIGSERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id),
    ip_address VARCHAR(45),
    user_agent TEXT,
    referer TEXT,
    country VARCHAR(100),
    device VARCHAR(100),
    browser VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
