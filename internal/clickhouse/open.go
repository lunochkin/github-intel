package clickhouse

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	clickhousestd "github.com/ClickHouse/clickhouse-go/v2"
	driverch "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Open connects to ClickHouse using CLICKHOUSE_HOST, CLICKHOUSE_PORT,
// CLICKHOUSE_DATABASE, CLICKHOUSE_USER, and CLICKHOUSE_PASSWORD.
func Open(ctx context.Context) (driverch.Conn, error) {
	host := os.Getenv("CLICKHOUSE_HOST")
	portStr := os.Getenv("CLICKHOUSE_PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("CLICKHOUSE_PORT: invalid %q", portStr)
	}

	opts := &clickhousestd.Options{
		Addr: []string{net.JoinHostPort(host, strconv.Itoa(port))},
		Auth: clickhousestd.Auth{
			Database: os.Getenv("CLICKHOUSE_DATABASE"),
			Username: os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},
	}

	conn, err := clickhousestd.Open(opts)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return conn, nil
}
