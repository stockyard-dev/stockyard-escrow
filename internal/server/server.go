package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-escrow/internal/store"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{
		db:      db,
		mux:     http.NewServeMux(),
		limits:  limits,
		dataDir: dataDir,
	}
	s.loadPersonalConfig()

	// Workflows
	s.mux.HandleFunc("GET /api/workflows", s.listWorkflows)
	s.mux.HandleFunc("POST /api/workflows", s.createWorkflow)
	s.mux.HandleFunc("GET /api/workflows/{id}", s.getWorkflow)
	s.mux.HandleFunc("PUT /api/workflows/{id}", s.updateWorkflow) // NEW
	s.mux.HandleFunc("DELETE /api/workflows/{id}", s.deleteWorkflow)

	// Requests
	s.mux.HandleFunc("GET /api/requests", s.listRequests)
	s.mux.HandleFunc("POST /api/requests", s.submitRequest)
	s.mux.HandleFunc("GET /api/requests/{id}", s.getRequest)
	s.mux.HandleFunc("PUT /api/requests/{id}", s.updateRequest)    // NEW
	s.mux.HandleFunc("DELETE /api/requests/{id}", s.deleteRequest) // NEW
	s.mux.HandleFunc("POST /api/requests/{id}/approve", s.approve)
	s.mux.HandleFunc("POST /api/requests/{id}/reject", s.reject)

	// Stats / health
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)

	// Personalization
	s.mux.HandleFunc("GET /api/config", s.configHandler)

	// Extras
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)

	// Tier
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"tier":        s.limits.Tier,
			"upgrade_url": "https://stockyard.dev/escrow/",
		})
	})

	// Dashboard
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ─── helpers ──────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusFound)
}

// ─── personalization ──────────────────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("escrow: warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("escrow: loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		writeJSON(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// ─── extras ───────────────────────────────────────────────────────

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	writeJSON(w, 200, out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		writeErr(w, 500, "save failed")
		return
	}
	writeJSON(w, 200, map[string]string{"ok": "saved"})
}

// ─── workflows ────────────────────────────────────────────────────

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"workflows": orEmpty(s.db.ListWorkflows())})
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf store.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if wf.Name == "" {
		writeErr(w, 400, "name required")
		return
	}
	if err := s.db.CreateWorkflow(&wf); err != nil {
		writeErr(w, 500, "create failed")
		return
	}
	writeJSON(w, 201, s.db.GetWorkflow(wf.ID))
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	wf := s.db.GetWorkflow(r.PathValue("id"))
	if wf == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, wf)
}

// updateWorkflow accepts a partial workflow. The original had no
// update endpoint at all.
func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ex := s.db.GetWorkflow(id)
	if ex == nil {
		writeErr(w, 404, "not found")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}

	patch := *ex
	if v, ok := raw["name"]; ok {
		var s string
		json.Unmarshal(v, &s)
		if s != "" {
			patch.Name = s
		}
	}
	if v, ok := raw["description"]; ok {
		json.Unmarshal(v, &patch.Description)
	}
	if v, ok := raw["approvers"]; ok {
		var arr []string
		if err := json.Unmarshal(v, &arr); err == nil {
			patch.Approvers = arr
		}
	}
	if v, ok := raw["require_all"]; ok {
		json.Unmarshal(v, &patch.RequireAll)
	}

	if err := s.db.UpdateWorkflow(id, &patch); err != nil {
		writeErr(w, 500, "update failed")
		return
	}
	writeJSON(w, 200, s.db.GetWorkflow(id))
}

func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Cascade extras for all requests in this workflow
	for _, rid := range s.db.WorkflowRequestIDs(id) {
		s.db.DeleteExtras("requests", rid)
	}
	if err := s.db.DeleteWorkflow(id); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.db.DeleteExtras("workflows", id)
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

// ─── requests ─────────────────────────────────────────────────────

func (s *Server) listRequests(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"requests": orEmpty(s.db.ListRequests(
			r.URL.Query().Get("workflow_id"),
			r.URL.Query().Get("status"),
		)),
	})
}

func (s *Server) submitRequest(w http.ResponseWriter, r *http.Request) {
	var req store.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if req.Title == "" || req.WorkflowID == "" {
		writeErr(w, 400, "title and workflow_id required")
		return
	}
	// Validate workflow exists
	if s.db.GetWorkflow(req.WorkflowID) == nil {
		writeErr(w, 400, "workflow not found")
		return
	}
	if err := s.db.SubmitRequest(&req); err != nil {
		writeErr(w, 500, "submit failed")
		return
	}
	writeJSON(w, 201, s.db.GetRequest(req.ID))
}

func (s *Server) getRequest(w http.ResponseWriter, r *http.Request) {
	req := s.db.GetRequest(r.PathValue("id"))
	if req == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, req)
}

// updateRequest accepts a partial payload for editable fields. Status
// and decisions are managed by the approve/reject endpoints. The
// original had no update endpoint for requests at all.
func (s *Server) updateRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ex := s.db.GetRequest(id)
	if ex == nil {
		writeErr(w, 404, "not found")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}

	patch := *ex
	if v, ok := raw["title"]; ok {
		var s string
		json.Unmarshal(v, &s)
		if s != "" {
			patch.Title = s
		}
	}
	if v, ok := raw["body"]; ok {
		json.Unmarshal(v, &patch.Body)
	}
	if v, ok := raw["submitter"]; ok {
		json.Unmarshal(v, &patch.Submitter)
	}

	if err := s.db.UpdateRequest(id, &patch); err != nil {
		writeErr(w, 500, "update failed")
		return
	}
	writeJSON(w, 200, s.db.GetRequest(id))
}

func (s *Server) deleteRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeleteRequest(id); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	s.db.DeleteExtras("requests", id)
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Approver string `json:"approver"`
		Comment  string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if err := s.db.Decide(r.PathValue("id"), d.Approver, "approved", d.Comment); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, s.db.GetRequest(r.PathValue("id")))
}

func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Approver string `json:"approver"`
		Comment  string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeErr(w, 400, "invalid json")
		return
	}
	if err := s.db.Decide(r.PathValue("id"), d.Approver, "rejected", d.Comment); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, s.db.GetRequest(r.PathValue("id")))
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.db.Stats())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	writeJSON(w, 200, map[string]any{
		"status":    "ok",
		"service":   "escrow",
		"workflows": st.Workflows,
		"pending":   st.Pending,
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
