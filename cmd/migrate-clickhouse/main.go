package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/lunochkin/github-intel/internal/dotenv"
)

func main() {
	log.SetFlags(0)
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	migrationsPath := flag.String("path", "migrations/clickhouse", "directory containing golang-migrate SQL files")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("usage: migrate-clickhouse [-path dir] <up|down|version|steps> [n]")
	}

	abs, err := filepath.Abs(*migrationsPath)
	if err != nil {
		log.Fatalf("migrations path: %v", err)
	}
	sourceURL := fmt.Sprintf("file://%s", filepath.ToSlash(abs))

	m, err := migrate.New(sourceURL, dsn())
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer m.Close()

	cmd := args[0]
	switch cmd {
	case "up":
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			err = nil
		}
	case "down":
		err = m.Steps(-1)
	case "version":
		var v uint
		var dirty bool
		v, dirty, err = m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("version: nil (no migrations applied)")
				return
			}
			break
		}
		log.Printf("version=%d dirty=%v", v, dirty)
		return
	case "steps":
		if len(args) < 2 {
			log.Fatal("steps requires an integer delta, e.g. steps 2 or steps -1")
		}
		var n int
		n, err = strconv.Atoi(args[1])
		if err != nil {
			log.Fatalf("steps n: %v", err)
		}
		err = m.Steps(n)
	default:
		log.Fatalf("unknown command %q", cmd)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(err)
	}
}

func dsn() string {
	host := os.Getenv("CLICKHOUSE_HOST")
	port := os.Getenv("CLICKHOUSE_PORT")
	user := os.Getenv("CLICKHOUSE_USER")
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	db := os.Getenv("CLICKHOUSE_MIGRATIONS_DATABASE")

	u := url.URL{
		Scheme: "clickhouse",
		Host:   net.JoinHostPort(host, port),
	}
	q := url.Values{}
	q.Set("username", user)
	if password != "" {
		q.Set("password", password)
	}
	q.Set("database", db)
	q.Set("x-multi-statement", "true")
	q.Set("x-migrations-table-engine", "MergeTree")
	u.RawQuery = q.Encode()
	return u.String()
}
