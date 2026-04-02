package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Workflow struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Approvers   []string `json:"approvers"`
	RequireAll  bool     `json:"require_all"`
	CreatedAt   string   `json:"created_at"`
	RequestCount int     `json:"request_count"`
}

type Request struct {
	ID          string     `json:"id"`
	WorkflowID  string     `json:"workflow_id"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	Submitter   string     `json:"submitter"`
	Status      string     `json:"status"` // pending, approved, rejected
	Decisions   []Decision `json:"decisions"`
	CreatedAt   string     `json:"created_at"`
	ResolvedAt  string     `json:"resolved_at,omitempty"`
}

type Decision struct {
	Approver  string `json:"approver"`
	Action    string `json:"action"` // approved, rejected
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil { return nil, err }
	dsn := filepath.Join(dataDir, "escrow.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil { return nil, err }
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS workflows (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT DEFAULT '', approvers_json TEXT DEFAULT '[]', require_all INTEGER DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS requests (id TEXT PRIMARY KEY, workflow_id TEXT NOT NULL, title TEXT NOT NULL, body TEXT DEFAULT '', submitter TEXT DEFAULT '', status TEXT DEFAULT 'pending', decisions_json TEXT DEFAULT '[]', created_at TEXT DEFAULT (datetime('now')), resolved_at TEXT DEFAULT '')`,
		`CREATE INDEX IF NOT EXISTS idx_requests_workflow ON requests(workflow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status)`,
	} {
		if _, err := db.Exec(q); err != nil { return nil, fmt.Errorf("migrate: %w", err) }
	}
	return &DB{db: db}, nil
}
func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreateWorkflow(w *Workflow) error {
	w.ID = genID(); w.CreatedAt = now(); if w.Approvers == nil { w.Approvers = []string{} }
	aj, _ := json.Marshal(w.Approvers); ra := 0; if w.RequireAll { ra = 1 }
	_, err := d.db.Exec(`INSERT INTO workflows (id,name,description,approvers_json,require_all,created_at) VALUES (?,?,?,?,?,?)`, w.ID, w.Name, w.Description, string(aj), ra, w.CreatedAt)
	return err
}
func (d *DB) GetWorkflow(id string) *Workflow {
	var w Workflow; var aj string; var ra int
	if err := d.db.QueryRow(`SELECT id,name,description,approvers_json,require_all,created_at FROM workflows WHERE id=?`, id).Scan(&w.ID, &w.Name, &w.Description, &aj, &ra, &w.CreatedAt); err != nil { return nil }
	json.Unmarshal([]byte(aj), &w.Approvers); w.RequireAll = ra == 1
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE workflow_id=?`, id).Scan(&w.RequestCount)
	return &w
}
func (d *DB) ListWorkflows() []Workflow {
	rows, _ := d.db.Query(`SELECT id,name,description,approvers_json,require_all,created_at FROM workflows ORDER BY name`)
	if rows == nil { return nil }; defer rows.Close()
	var out []Workflow
	for rows.Next() {
		var w Workflow; var aj string; var ra int
		rows.Scan(&w.ID, &w.Name, &w.Description, &aj, &ra, &w.CreatedAt)
		json.Unmarshal([]byte(aj), &w.Approvers); w.RequireAll = ra == 1
		d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE workflow_id=?`, w.ID).Scan(&w.RequestCount)
		out = append(out, w)
	}
	return out
}
func (d *DB) DeleteWorkflow(id string) error { d.db.Exec(`DELETE FROM requests WHERE workflow_id=?`, id); _, err := d.db.Exec(`DELETE FROM workflows WHERE id=?`, id); return err }

func (d *DB) SubmitRequest(r *Request) error {
	r.ID = genID(); r.CreatedAt = now(); r.Status = "pending"; r.Decisions = []Decision{}
	dj, _ := json.Marshal(r.Decisions)
	_, err := d.db.Exec(`INSERT INTO requests (id,workflow_id,title,body,submitter,status,decisions_json,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		r.ID, r.WorkflowID, r.Title, r.Body, r.Submitter, r.Status, string(dj), r.CreatedAt)
	return err
}
func (d *DB) GetRequest(id string) *Request {
	var r Request; var dj string
	if err := d.db.QueryRow(`SELECT id,workflow_id,title,body,submitter,status,decisions_json,created_at,resolved_at FROM requests WHERE id=?`, id).Scan(&r.ID, &r.WorkflowID, &r.Title, &r.Body, &r.Submitter, &r.Status, &dj, &r.CreatedAt, &r.ResolvedAt); err != nil { return nil }
	json.Unmarshal([]byte(dj), &r.Decisions); return &r
}
func (d *DB) ListRequests(workflowID, status string) []Request {
	q := `SELECT id,workflow_id,title,body,submitter,status,decisions_json,created_at,resolved_at FROM requests WHERE 1=1`
	var args []any
	if workflowID != "" { q += ` AND workflow_id=?`; args = append(args, workflowID) }
	if status != "" && status != "all" { q += ` AND status=?`; args = append(args, status) }
	q += ` ORDER BY created_at DESC LIMIT 50`
	rows, _ := d.db.Query(q, args...)
	if rows == nil { return nil }; defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request; var dj string
		rows.Scan(&r.ID, &r.WorkflowID, &r.Title, &r.Body, &r.Submitter, &r.Status, &dj, &r.CreatedAt, &r.ResolvedAt)
		json.Unmarshal([]byte(dj), &r.Decisions); out = append(out, r)
	}
	return out
}
func (d *DB) Decide(requestID, approver, action, comment string) error {
	r := d.GetRequest(requestID); if r == nil { return fmt.Errorf("not found") }
	if r.Status != "pending" { return fmt.Errorf("already resolved") }
	dec := Decision{Approver: approver, Action: action, Comment: comment, CreatedAt: now()}
	r.Decisions = append(r.Decisions, dec)
	dj, _ := json.Marshal(r.Decisions)
	w := d.GetWorkflow(r.WorkflowID)
	// Determine if resolved
	if action == "rejected" { r.Status = "rejected"; r.ResolvedAt = now() } else if w != nil {
		if w.RequireAll {
			if len(r.Decisions) >= len(w.Approvers) { r.Status = "approved"; r.ResolvedAt = now() }
		} else { r.Status = "approved"; r.ResolvedAt = now() }
	}
	_, err := d.db.Exec(`UPDATE requests SET status=?,decisions_json=?,resolved_at=? WHERE id=?`, r.Status, string(dj), r.ResolvedAt, requestID)
	return err
}

type Stats struct { Workflows int `json:"workflows"`; Pending int `json:"pending"`; Approved int `json:"approved"`; Rejected int `json:"rejected"` }
func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM workflows`).Scan(&s.Workflows)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='pending'`).Scan(&s.Pending)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='approved'`).Scan(&s.Approved)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='rejected'`).Scan(&s.Rejected)
	return s
}
