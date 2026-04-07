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

// Workflow defines an approval flow with a list of approvers and a
// require-all flag. If RequireAll is false, the first non-rejection
// approval resolves the request.
type Workflow struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Approvers    []string `json:"approvers"`
	RequireAll   bool     `json:"require_all"`
	CreatedAt    string   `json:"created_at"`
	RequestCount int      `json:"request_count"`
}

// Request is a single approval request running through a Workflow.
// Status is one of: pending, approved, rejected.
type Request struct {
	ID         string     `json:"id"`
	WorkflowID string     `json:"workflow_id"`
	Title      string     `json:"title"`
	Body       string     `json:"body,omitempty"`
	Submitter  string     `json:"submitter"`
	Status     string     `json:"status"`
	Decisions  []Decision `json:"decisions"`
	CreatedAt  string     `json:"created_at"`
	ResolvedAt string     `json:"resolved_at,omitempty"`
}

// Decision is a single vote by one approver on one request.
type Decision struct {
	Approver  string `json:"approver"`
	Action    string `json:"action"` // approved, rejected
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"created_at"`
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
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			approvers_json TEXT DEFAULT '[]',
			require_all INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,
			workflow_id TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT DEFAULT '',
			submitter TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			decisions_json TEXT DEFAULT '[]',
			created_at TEXT DEFAULT (datetime('now')),
			resolved_at TEXT DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_workflow ON requests(workflow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status)`,
		`CREATE TABLE IF NOT EXISTS extras(
			resource TEXT NOT NULL,
			record_id TEXT NOT NULL,
			data TEXT NOT NULL DEFAULT '{}',
			PRIMARY KEY(resource, record_id)
		)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string   { return time.Now().UTC().Format(time.RFC3339) }

// ─── Workflows ────────────────────────────────────────────────────

func (d *DB) CreateWorkflow(w *Workflow) error {
	w.ID = genID()
	w.CreatedAt = now()
	if w.Approvers == nil {
		w.Approvers = []string{}
	}
	aj, _ := json.Marshal(w.Approvers)
	ra := 0
	if w.RequireAll {
		ra = 1
	}
	_, err := d.db.Exec(
		`INSERT INTO workflows (id, name, description, approvers_json, require_all, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, w.Description, string(aj), ra, w.CreatedAt,
	)
	return err
}

func (d *DB) GetWorkflow(id string) *Workflow {
	var w Workflow
	var aj string
	var ra int
	err := d.db.QueryRow(
		`SELECT id, name, description, approvers_json, require_all, created_at
		 FROM workflows WHERE id=?`,
		id,
	).Scan(&w.ID, &w.Name, &w.Description, &aj, &ra, &w.CreatedAt)
	if err != nil {
		return nil
	}
	json.Unmarshal([]byte(aj), &w.Approvers)
	if w.Approvers == nil {
		w.Approvers = []string{}
	}
	w.RequireAll = ra == 1
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE workflow_id=?`, id).Scan(&w.RequestCount)
	return &w
}

func (d *DB) ListWorkflows() []Workflow {
	rows, _ := d.db.Query(
		`SELECT id, name, description, approvers_json, require_all, created_at
		 FROM workflows ORDER BY name ASC`,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Workflow
	for rows.Next() {
		var w Workflow
		var aj string
		var ra int
		rows.Scan(&w.ID, &w.Name, &w.Description, &aj, &ra, &w.CreatedAt)
		json.Unmarshal([]byte(aj), &w.Approvers)
		if w.Approvers == nil {
			w.Approvers = []string{}
		}
		w.RequireAll = ra == 1
		d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE workflow_id=?`, w.ID).Scan(&w.RequestCount)
		out = append(out, w)
	}
	return out
}

// UpdateWorkflow rewrites mutable workflow fields. The original
// implementation had no UpdateWorkflow at all — workflows were
// create+delete-only.
func (d *DB) UpdateWorkflow(id string, w *Workflow) error {
	aj, _ := json.Marshal(w.Approvers)
	ra := 0
	if w.RequireAll {
		ra = 1
	}
	_, err := d.db.Exec(
		`UPDATE workflows SET name=?, description=?, approvers_json=?, require_all=? WHERE id=?`,
		w.Name, w.Description, string(aj), ra, id,
	)
	return err
}

// DeleteWorkflow removes the workflow and all its requests. Caller is
// responsible for clearing extras.
func (d *DB) DeleteWorkflow(id string) error {
	d.db.Exec(`DELETE FROM requests WHERE workflow_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM workflows WHERE id=?`, id)
	return err
}

// WorkflowRequestIDs lists every request belonging to a workflow.
// Used by the cascade delete in the server to clean up extras.
func (d *DB) WorkflowRequestIDs(workflowID string) []string {
	rows, _ := d.db.Query(`SELECT id FROM requests WHERE workflow_id=?`, workflowID)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

// ─── Requests ─────────────────────────────────────────────────────

func (d *DB) SubmitRequest(r *Request) error {
	r.ID = genID()
	r.CreatedAt = now()
	r.Status = "pending"
	r.Decisions = []Decision{}
	dj, _ := json.Marshal(r.Decisions)
	_, err := d.db.Exec(
		`INSERT INTO requests (id, workflow_id, title, body, submitter, status, decisions_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.WorkflowID, r.Title, r.Body, r.Submitter, r.Status, string(dj), r.CreatedAt,
	)
	return err
}

func (d *DB) GetRequest(id string) *Request {
	var r Request
	var dj string
	err := d.db.QueryRow(
		`SELECT id, workflow_id, title, body, submitter, status, decisions_json, created_at, resolved_at
		 FROM requests WHERE id=?`,
		id,
	).Scan(&r.ID, &r.WorkflowID, &r.Title, &r.Body, &r.Submitter, &r.Status, &dj, &r.CreatedAt, &r.ResolvedAt)
	if err != nil {
		return nil
	}
	json.Unmarshal([]byte(dj), &r.Decisions)
	if r.Decisions == nil {
		r.Decisions = []Decision{}
	}
	return &r
}

func (d *DB) ListRequests(workflowID, status string) []Request {
	q := `SELECT id, workflow_id, title, body, submitter, status, decisions_json, created_at, resolved_at
	      FROM requests WHERE 1=1`
	var args []any
	if workflowID != "" {
		q += ` AND workflow_id=?`
		args = append(args, workflowID)
	}
	if status != "" && status != "all" {
		q += ` AND status=?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, _ := d.db.Query(q, args...)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		var dj string
		rows.Scan(&r.ID, &r.WorkflowID, &r.Title, &r.Body, &r.Submitter, &r.Status, &dj, &r.CreatedAt, &r.ResolvedAt)
		json.Unmarshal([]byte(dj), &r.Decisions)
		if r.Decisions == nil {
			r.Decisions = []Decision{}
		}
		out = append(out, r)
	}
	return out
}

// UpdateRequest rewrites the editable request fields (title, body,
// submitter). Status and decisions are managed by Decide. The original
// implementation had no UpdateRequest at all.
func (d *DB) UpdateRequest(id string, r *Request) error {
	_, err := d.db.Exec(
		`UPDATE requests SET title=?, body=?, submitter=? WHERE id=?`,
		r.Title, r.Body, r.Submitter, id,
	)
	return err
}

func (d *DB) DeleteRequest(id string) error {
	_, err := d.db.Exec(`DELETE FROM requests WHERE id=?`, id)
	return err
}

// Decide casts a vote on a request. Validates that:
//   - the request exists and is still pending
//   - the workflow exists and the approver is in its approver list
//   - the same approver hasn't already voted on this request
//   - the action is one of "approved" or "rejected"
//
// Resolution rules: any rejection immediately rejects the request.
// Otherwise approvals resolve when require_all is false (first approval
// wins) or when all listed approvers have approved (require_all=true).
func (d *DB) Decide(requestID, approver, action, comment string) error {
	if action != "approved" && action != "rejected" {
		return fmt.Errorf("action must be approved or rejected")
	}
	if approver == "" {
		return fmt.Errorf("approver required")
	}

	r := d.GetRequest(requestID)
	if r == nil {
		return fmt.Errorf("request not found")
	}
	if r.Status != "pending" {
		return fmt.Errorf("request already resolved")
	}

	w := d.GetWorkflow(r.WorkflowID)
	if w == nil {
		return fmt.Errorf("workflow not found")
	}

	// Approver must be in the workflow's approver list
	allowed := false
	for _, a := range w.Approvers {
		if a == approver {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("approver not authorized for this workflow")
	}

	// Same approver can't vote twice
	for _, dec := range r.Decisions {
		if dec.Approver == approver {
			return fmt.Errorf("approver has already voted")
		}
	}

	r.Decisions = append(r.Decisions, Decision{
		Approver:  approver,
		Action:    action,
		Comment:   comment,
		CreatedAt: now(),
	})

	// Resolution
	if action == "rejected" {
		r.Status = "rejected"
		r.ResolvedAt = now()
	} else if w.RequireAll {
		// Count distinct approvals
		approvals := 0
		for _, dec := range r.Decisions {
			if dec.Action == "approved" {
				approvals++
			}
		}
		if approvals >= len(w.Approvers) {
			r.Status = "approved"
			r.ResolvedAt = now()
		}
	} else {
		r.Status = "approved"
		r.ResolvedAt = now()
	}

	dj, _ := json.Marshal(r.Decisions)
	_, err := d.db.Exec(
		`UPDATE requests SET status=?, decisions_json=?, resolved_at=? WHERE id=?`,
		r.Status, string(dj), r.ResolvedAt, requestID,
	)
	return err
}

// ─── Stats ────────────────────────────────────────────────────────

type Stats struct {
	Workflows int `json:"workflows"`
	Requests  int `json:"requests"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Rejected  int `json:"rejected"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM workflows`).Scan(&s.Workflows)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests`).Scan(&s.Requests)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='pending'`).Scan(&s.Pending)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='approved'`).Scan(&s.Approved)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE status='rejected'`).Scan(&s.Rejected)
	return s
}

// ─── Extras ───────────────────────────────────────────────────────

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
