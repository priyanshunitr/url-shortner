package database

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func PostgreSQLConnection() (*pgxpool.Pool, error) {
	//dburl defn
	dbURL := os.Getenv("DB_SERVER_URL")
	conf, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	//fetch pool settings from env
	maxConn, err := strconv.Atoi(os.Getenv("DB_MAX_CONNECTIONS"))
	if err != nil || maxConn <= 0 {
		maxConn = 20
	}
	minConn, err := strconv.Atoi(os.Getenv("DB_MIN_CONNECTIONS"))
	if err != nil || minConn <= 0 {
		minConn = 5
	}
	maxIdleConnTime, err := strconv.Atoi(os.Getenv("DB_MAX_IDLETIME_CONNECTION_TIME"))
	if err != nil || maxIdleConnTime <= 0 {
		maxIdleConnTime = 10
	}
	maxLifetimeConnTime, err := strconv.Atoi(os.Getenv("DB_MAX_LIFETIME_CONNECTION_TIME"))
	if err != nil || maxLifetimeConnTime <= 0 {
		maxLifetimeConnTime = 1
	}

	//pool settings
	conf.MaxConns = int32(maxConn)
	conf.MinConns = int32(minConn)
	conf.MaxConnLifetime = time.Duration(maxLifetimeConnTime) * time.Hour
	conf.MaxConnIdleTime = time.Duration(maxIdleConnTime) * time.Minute

	//define db connection for postgreSQL
	db, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	//verify db connection
	if err := db.Ping(context.Background()); err != nil {
		return nil, err
	}

	log.Println("db connection successful")
	log.Println(os.Getenv("DB_SERVER_URL"))
	return db, nil
}
