package migrations

// users
// (
//     id BIGSERIAL PRIMARY KEY,
//     email VARCHAR(255) UNIQUE NOT NULL,
//     password_hash TEXT NOT NULL,
//     created_at TIMESTAMP NOT NULL
// )

// urls
// (
//     id BIGSERIAL PRIMARY KEY,
//     user_id BIGINT REFERENCES users(id),
//     long_url TEXT,
//     short_url TEXT,
//     expiry TIMESTAMP,
//     clicks BIGINT,
//     created_at TIMESTAMP
// )

// click_events
// (
//     id BIGSERIAL PRIMARY KEY,
//     url_id BIGINT REFERENCES urls(id),
//     ip_address TEXT,
//     user_agent TEXT,
//     referer TEXT,
//     country TEXT,
//     device TEXT,
//     browser TEXT,
//     created_at TIMESTAMP
// )
