package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/SrVariable/SPL/config"
	_ "github.com/go-sql-driver/mysql"
)

//go:embed schema.sql
var schema string

// Connect opens the pool and verifies the server is actually reachable, so a
// forgotten `docker compose up -d` fails here instead of on the first query.
func Connect(cfg config.DBConfig) (*sql.DB, error) {
	database, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("couldn't reach the database at %s:%s: %w", cfg.Host, cfg.Port, err)
	}

	return database, nil
}

// splitStatements turns the embedded schema into individual statements. The
// schema only uses ';' as a statement terminator, so a plain split is enough
// once the comment lines are gone.
func splitStatements(sqlText string) []string {
	var stripped strings.Builder
	for line := range strings.Lines(sqlText) {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		stripped.WriteString(line)
	}

	var statements []string
	for _, statement := range strings.Split(stripped.String(), ";") {
		if statement = strings.TrimSpace(statement); statement != "" {
			statements = append(statements, statement)
		}
	}

	return statements
}

// Migrate applies schema.sql. Every statement is idempotent, so it's safe to
// call on every start.
func Migrate(ctx context.Context, database *sql.DB) error {
	for _, statement := range splitStatements(schema) {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("failed to apply schema statement %q: %w", firstLine(statement), err)
		}
	}

	return nil
}

func firstLine(statement string) string {
	line, _, _ := strings.Cut(statement, "\n")
	return line
}
