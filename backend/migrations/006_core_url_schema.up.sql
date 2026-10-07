-- Preserve existing URL records while adopting the core API's column names.
ALTER TABLE urls RENAME COLUMN short_url TO short_code;
ALTER TABLE urls RENAME COLUMN long_url TO original_url;
ALTER TABLE urls RENAME COLUMN clicks TO click_count;
ALTER TABLE urls RENAME COLUMN expiry TO expires_at;
ALTER TABLE urls ALTER COLUMN short_code TYPE VARCHAR(16);
ALTER TABLE urls ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE urls ALTER COLUMN expires_at TYPE TIMESTAMPTZ USING expires_at AT TIME ZONE 'UTC';
UPDATE urls SET click_count = 0 WHERE click_count IS NULL;
ALTER TABLE urls ALTER COLUMN click_count SET NOT NULL;
ALTER TABLE urls ADD COLUMN last_accessed TIMESTAMPTZ;
ALTER TABLE click_events ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE click_events DROP CONSTRAINT click_events_url_id_fkey;
ALTER TABLE click_events ADD CONSTRAINT click_events_url_id_fkey FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_click_events_url_created ON click_events(url_id, created_at);
