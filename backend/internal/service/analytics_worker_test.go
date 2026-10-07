package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gottatouchsomegrass/url/internal/cache"
	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type batchRepoStub struct {
	BatchRepository
	failed bool
	ledger map[string]bool
	clicks int64
}
type batchTxStub struct {
	pgx.Tx
	repo  *batchRepoStub
	batch *models.AnalyticsBatch
}

func (r *batchRepoStub) Begin(context.Context) (pgx.Tx, error) {
	if r.failed {
		return nil, errors.New("database unavailable")
	}
	return &batchTxStub{repo: r}, nil
}
func (r *batchRepoStub) ApplyBatchTx(_ context.Context, tx pgx.Tx, batch *models.AnalyticsBatch) error {
	tx.(*batchTxStub).batch = batch
	return nil
}
func (tx *batchTxStub) Commit(context.Context) error {
	if !tx.repo.ledger[tx.batch.ID] {
		for _, c := range tx.batch.Counts {
			tx.repo.clicks += c.Count
		}
		tx.repo.ledger[tx.batch.ID] = true
	}
	return nil
}
func (tx *batchTxStub) Rollback(context.Context) error { return nil }

type ackFailure struct {
	AnalyticsBuffer
	failed bool
}

func (b *ackFailure) Ack(ctx context.Context, key string) error {
	if b.failed {
		return errors.New("lost Redis ACK")
	}
	return b.AnalyticsBuffer.Ack(ctx, key)
}

func TestAnalyticsSurvivesDatabaseFailureAndLostAcknowledgement(t *testing.T) {
	m := miniredis.RunT(t)
	m.SetTime(time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC))
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	buffer := &cache.RedisAnalytics{Client: client}
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		stored, err := buffer.Record(ctx, &models.ClickEvent{URLID: 12, CreatedAt: time.Now().UTC()})
		if err != nil || !stored {
			t.Fatalf("record: %t, %v", stored, err)
		}
	}
	repo := &batchRepoStub{failed: true, ledger: map[string]bool{}}
	acks := &ackFailure{AnalyticsBuffer: buffer}
	worker := &AnalyticsWorker{Buffer: acks, Repo: repo}
	if err := worker.Flush(ctx); err == nil {
		t.Fatal("database error was hidden")
	}
	keys, _ := buffer.Keys(ctx)
	if len(keys) != 1 || repo.clicks != 0 {
		t.Fatal("failed batch was discarded")
	}
	batch, err := buffer.Load(ctx, keys[0])
	if err != nil || batch.Counts[12].Count != 100 || len(batch.Events) != 100 || batch.Counts[12].LastAccessed.IsZero() {
		t.Fatalf("incorrect pending batch: %+v, %v", batch, err)
	}
	repo.failed, acks.failed = false, true
	if err := worker.Flush(ctx); err == nil || repo.clicks != 100 {
		t.Fatalf("lost ACK: clicks %d, %v", repo.clicks, err)
	}
	// A fresh worker can resume the persisted Redis batch after a process exit.
	acks.failed = false
	restarted := &AnalyticsWorker{Buffer: acks, Repo: repo}
	if err := restarted.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	keys, _ = buffer.Keys(ctx)
	if repo.clicks != 100 || len(keys) != 0 {
		t.Fatalf("retry double-counted: clicks %d, pending %v", repo.clicks, keys)
	}
}
