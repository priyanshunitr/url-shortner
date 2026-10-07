// Package database opens db conn
package database

import (
	"github.com/gottatouchsomegrass/url/app/repositories"
)

type Queries struct {
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
		return nil, err
	}

	return &Queries{
		URLQuery: &repositories.URLQuery{
			DB:  db,
			RDB: rdb,
		},
		AnalyticsQuery: &repositories.AnalyticsQuery{
			DB: db,
		},
		UserQuery: &repositories.UserQuery{
			DB: db,
		},
	}, nil
}
