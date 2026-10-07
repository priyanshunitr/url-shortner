package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gottatouchsomegrass/url/internal/model"
	"github.com/redis/go-redis/v9"
)

const analyticsPending = "analytics:{buffer}:pending"
const analyticsBatches = "analytics:{buffer}:batches"

type RedisAnalytics struct{ Client *redis.Client }

var recordClick = redis.NewScript(`
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
redis.call('HINCRBY', KEYS[1], 'c:' .. ARGV[1], 1)
redis.call('HSET', KEYS[1], 't:' .. ARGV[1], string.format('%.0f', now))
redis.call('ZINCRBY', KEYS[2], 1, ARGV[3])
redis.call('EXPIRE', KEYS[2], 172800)
local n = tonumber(redis.call('HGET', KEYS[1], 'n') or '0')
if n >= 10000 then return 0 end
redis.call('HSET', KEYS[1], 'e:' .. tostring(n), ARGV[2])
redis.call('HSET', KEYS[1], 'n', n + 1)
return 1
`)

var sealBatch = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return 0 end
if redis.call('EXISTS', KEYS[2]) == 1 then return redis.error_reply('batch ID collision') end
redis.call('RENAME', KEYS[1], KEYS[2])
redis.call('SADD', KEYS[3], KEYS[2])
return 1
`)

var ackBatch = redis.NewScript(`
redis.call('DEL', KEYS[1])
redis.call('SREM', KEYS[2], KEYS[1])
return 1
`)

// Record performs no PostgreSQL writes. Counts are exact; event details are
// bounded to 10,000 samples per pending batch during prolonged DB outages.
func (b *RedisAnalytics) Record(ctx context.Context, event *models.ClickEvent) (bool, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return false, err
	}
	hotKey := "analytics:{buffer}:hot:" + event.CreatedAt.UTC().Format("20060102")
	stored, err := recordClick.Run(ctx, b.Client, []string{analyticsPending, hotKey}, event.URLID, data, strconv.FormatInt(event.URLID, 10)).Int()
	return stored == 1, err
}

func (b *RedisAnalytics) Seal(ctx context.Context) error {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	key := "analytics:{buffer}:batch:" + hex.EncodeToString(id[:])
	return sealBatch.Run(ctx, b.Client, []string{analyticsPending, key, analyticsBatches}).Err()
}

func (b *RedisAnalytics) Keys(ctx context.Context) ([]string, error) {
	return b.Client.SMembers(ctx, analyticsBatches).Result()
}

func (b *RedisAnalytics) Load(ctx context.Context, key string) (*models.AnalyticsBatch, error) {
	fields, err := b.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	batch := &models.AnalyticsBatch{ID: key, Counts: map[int64]models.ClickAggregate{}}
	for field, value := range fields {
		if strings.HasPrefix(field, "e:") {
			var event models.ClickEvent
			if err := json.Unmarshal([]byte(value), &event); err != nil {
				return nil, err
			}
			batch.Events = append(batch.Events, event)
			continue
		}
		if field == "n" {
			continue
		}
		if len(field) < 3 {
			return nil, errors.New("invalid analytics field")
		}
		id, err := strconv.ParseInt(field[2:], 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("invalid analytics URL ID")
		}
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		count := batch.Counts[id]
		switch field[:2] {
		case "c:":
			count.Count = v
		case "t:":
			count.LastAccessed = time.UnixMilli(v).UTC()
		default:
			return nil, errors.New("invalid analytics field prefix")
		}
		batch.Counts[id] = count
	}
	return batch, nil
}

func (b *RedisAnalytics) Ack(ctx context.Context, key string) error {
	return ackBatch.Run(ctx, b.Client, []string{key, analyticsBatches}).Err()
}
