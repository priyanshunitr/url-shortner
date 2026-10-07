// Package database opens db conn
package database

import (
	"github.com/gottatouchsomegrass/url/internal/repository"
	"github.com/redis/go-redis/v9"
)

type Queries struct {
	Redis *redis.Client
	*repositories.URLQuery
	*repositories.AnalyticsQuery
	*repositories.UserQuery
}

func OpenDBConnection() (*Queries, error) {
	//postgres conn
	db, err := PostgreSQLConnection()
	if err != nil {
		return nil, err
	}

	//redis conn
	rdb, err := RedisConnection()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Queries{
		Redis: rdb,
		URLQuery: &repositories.URLQuery{
			DB: db,
		},
		AnalyticsQuery: &repositories.AnalyticsQuery{
			DB: db,
		},
		UserQuery: &repositories.UserQuery{
			DB:  db,
			RDB: rdb,
		},
	}, nil
}
