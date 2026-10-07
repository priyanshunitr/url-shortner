package repositories

import (
	"context"
	"sort"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5"
)

func (q *URLQuery) Begin(ctx context.Context) (pgx.Tx, error) { return q.DB.Begin(ctx) }

// ApplyBatchTx is idempotent: the ledger and all increments share one commit.
func (q *URLQuery) ApplyBatchTx(ctx context.Context, tx pgx.Tx, batch *models.AnalyticsBatch) error {
	tag, err := tx.Exec(ctx, `INSERT INTO analytics_batches(id) VALUES ($1) ON CONFLICT DO NOTHING`, batch.ID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	ids := make([]int64, 0, len(batch.Counts))
	for id := range batch.Counts {
		ids = append(ids, id)
	}
	// Consistent lock order avoids deadlocks between independent flush workers.
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	queries := &pgx.Batch{}
	for _, id := range ids {
		count := batch.Counts[id]
		queries.Queue(`UPDATE urls SET click_count = click_count + $2,
			last_accessed = GREATEST(last_accessed, $3::timestamptz) WHERE id = $1`, id, count.Count, count.LastAccessed)
	}
	for _, e := range batch.Events {
		// A URL deleted before flush should not poison every other URL's batch.
		queries.Queue(`INSERT INTO click_events(url_id, ip_address, user_agent, referer, country, device, browser, created_at)
			SELECT id, $2, $3, $4, $5, $6, $7, $8 FROM urls WHERE id = $1`, e.URLID, e.IPAddress, e.UserAgent, e.Referer, e.Country, e.Device, e.Browser, e.CreatedAt)
	}
	return tx.SendBatch(ctx, queries).Close()
}
