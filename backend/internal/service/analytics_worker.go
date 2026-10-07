package services

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gottatouchsomegrass/url/internal/metrics"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5"
)

type AnalyticsBuffer interface {
	Seal(context.Context) error
	Keys(context.Context) ([]string, error)
	Load(context.Context, string) (*models.AnalyticsBatch, error)
	Ack(context.Context, string) error
}

type BatchRepository interface {
	Begin(context.Context) (pgx.Tx, error)
	ApplyBatchTx(context.Context, pgx.Tx, *models.AnalyticsBatch) error
}

type AnalyticsWorker struct {
	Buffer   AnalyticsBuffer
	Repo     BatchRepository
	Interval time.Duration
	Metrics  *metrics.Metrics
	mu       sync.Mutex
}

func (w *AnalyticsWorker) Flush(ctx context.Context) (result error) {
	defer func() {
		if result != nil && w.Metrics != nil {
			w.Metrics.FlushErrors.Inc()
		}
	}()
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.Buffer.Seal(ctx); err != nil {
		return err
	}
	keys, err := w.Buffer.Keys(ctx)
	if err != nil {
		return err
	}
	if w.Metrics != nil {
		w.Metrics.PendingBatches.Set(float64(len(keys)))
	}
	var failures error
	for _, key := range keys {
		if err := w.persist(ctx, key); err != nil {
			failures = errors.Join(failures, err)
		} else if w.Metrics != nil {
			w.Metrics.FlushedBatches.Inc()
			w.Metrics.PendingBatches.Dec()
		}
	}
	return failures
}

func (w *AnalyticsWorker) persist(ctx context.Context, key string) error {
	batch, err := w.Buffer.Load(ctx, key)
	if err != nil {
		return err
	}
	if len(batch.Counts) == 0 {
		return w.Buffer.Ack(ctx, key)
	}
	tx, err := w.Repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := w.Repo.ApplyBatchTx(ctx, tx, batch); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// ACK only after the durable commit; a lost ACK is safe to retry.
	return w.Buffer.Ack(ctx, key)
}

func (w *AnalyticsWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flushCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			if err := w.Flush(flushCtx); err != nil {
				log.Printf("analytics flush failed; retained for retry: %v", err)
			}
			cancel()
		}
	}
}
