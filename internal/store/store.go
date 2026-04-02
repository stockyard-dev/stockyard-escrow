package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Workflow struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Approver     string   `json:"approver"`
	CreatedAt    string   `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "escrow.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,\n\t\t\tname TEXT DEFAULT '',\n\t\t\tdescription TEXT DEFAULT '',\n\t\t\tstatus TEXT DEFAULT 'pending',\n\t\t\tapprover TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func (d *DB) Create(e *Workflow) error {
	e.ID = genID()
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`INSERT INTO workflows (id, name, description, status, approver, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Description, e.Status, e.Approver, e.CreatedAt)
	return err
}

func (d *DB) Get(id string) *Workflow {
	row := d.db.QueryRow(`SELECT id, name, description, status, approver, created_at FROM workflows WHERE id=?`, id)
	var e Workflow
	if err := row.Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.Approver, &e.CreatedAt); err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Workflow {
	rows, err := d.db.Query(`SELECT id, name, description, status, approver, created_at FROM workflows ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Workflow
	for rows.Next() {
		var e Workflow
		if err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Status, &e.Approver, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM workflows WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM workflows`).Scan(&n)
	return n
}
