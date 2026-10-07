package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.up.sql *.down.sql
var Files embed.FS

// Run applies each migration transactionally using the standard
// schema_migrations(version, dirty) table, compatible with the migrate CLI.
func Run(ctx context.Context, pool *pgxpool.Pool, direction string, steps int) error {
	if (direction != "up" && direction != "down") || steps < 0 {
		return errors.New("invalid migration direction/steps")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(8760123401)`); err != nil {
		return err
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(8760123401)`) }()
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)`); err != nil {
		return err
	}
	var version int
	var dirty bool
	err = conn.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if dirty {
		return fmt.Errorf("migration %d is marked dirty; repair it before proceeding", version)
	}
	files, err := Files.ReadDir(".")
	if err != nil {
		return err
	}
	versions := map[int]string{}
	var order []int
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "."+direction+".sql") {
			continue
		}
		n, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			return err
		}
		if (direction == "up" && n > version) || (direction == "down" && n <= version) {
			versions[n] = file.Name()
			order = append(order, n)
		}
	}
	sort.Ints(order)
	if direction == "down" {
		sort.Sort(sort.Reverse(sort.IntSlice(order)))
	}
	if steps > 0 && len(order) > steps {
		order = order[:steps]
	}
	for _, n := range order {
		source, err := Files.ReadFile(versions[n])
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		next := n
		if direction == "down" {
			next = n - 1
		}
		err = func() error {
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err := tx.Exec(ctx, string(source)); err != nil {
				return fmt.Errorf("migration %s: %w", versions[n], err)
			}
			if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations`); err != nil {
				return err
			}
			if next > 0 {
				if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version, dirty) VALUES ($1, false)`, next); err != nil {
					return err
				}
			}
			return tx.Commit(ctx)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
