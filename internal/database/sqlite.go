package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenSQLite opens (or creates) the SQLite database in WAL mode and runs embedded migrations.
func OpenSQLite(dbPath string) (*sql.DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Single writer connection prevents SQLITE_BUSY under concurrent Go requests.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec pragma %q: %w", p, err)
		}
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := ensureCommercialColumns(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func ensureCommercialColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(contracts);`)
	if err != nil {
		return fmt.Errorf("inspect contracts table: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan table_info: %w", err)
		}
		existing[name] = true
	}

	additions := []struct {
		col string
		ddl string
	}{
		{"has_monthly_fee", "ALTER TABLE contracts ADD COLUMN has_monthly_fee INTEGER NOT NULL DEFAULT 1;"},
		{"setup_installments", "ALTER TABLE contracts ADD COLUMN setup_installments INTEGER NOT NULL DEFAULT 1;"},
		{"setup_payment_method", "ALTER TABLE contracts ADD COLUMN setup_payment_method TEXT NOT NULL DEFAULT 'pix';"},
		{"setup_total_final_cents", "ALTER TABLE contracts ADD COLUMN setup_total_final_cents INTEGER NOT NULL DEFAULT 0;"},
		{"setup_installment_value_cents", "ALTER TABLE contracts ADD COLUMN setup_installment_value_cents INTEGER NOT NULL DEFAULT 0;"},
		{"pay_on_signature", "ALTER TABLE contracts ADD COLUMN pay_on_signature INTEGER NOT NULL DEFAULT 1;"},
		{"include_witnesses", "ALTER TABLE contracts ADD COLUMN include_witnesses INTEGER NOT NULL DEFAULT 1;"},
		{"modelo_contrato", "ALTER TABLE contracts ADD COLUMN modelo_contrato TEXT NOT NULL DEFAULT 'padrao';"},
		{"escopo_json", "ALTER TABLE contracts ADD COLUMN escopo_json TEXT NOT NULL DEFAULT '{}';"},
	}
	for _, a := range additions {
		if !existing[a.col] {
			if _, err := db.Exec(a.ddl); err != nil {
				return fmt.Errorf("add column %s: %w", a.col, err)
			}
		}
	}
	return nil
}

func runMigrations(db *sql.DB) error {
	sqlBytes, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration 001_init.sql: %w", err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec migration 001_init.sql: %w", err)
	}
	return nil
}
