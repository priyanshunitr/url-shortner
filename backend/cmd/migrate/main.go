package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/gottatouchsomegrass/url/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	direction := flag.String("direction", "up", "up or down")
	steps := flag.Int("steps", 0, "number of migrations; zero means all")
	flag.Parse()
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)
	if os.Getenv("DB_SERVER_URL") == "" {
		log.Fatal("DB_SERVER_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_SERVER_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Run(ctx, pool, *direction, *steps); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations complete")
}
