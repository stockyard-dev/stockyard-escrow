package server
import ("encoding/json";"log";"net/http";"github.com/stockyard-dev/stockyard-escrow/internal/store")
type Server struct { db *store.DB; mux *http.ServeMux; limits Limits }
func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}
	s.mux.HandleFunc("GET /api/workflows", s.listWorkflows)
	s.mux.HandleFunc("POST /api/workflows", s.createWorkflow)
	s.mux.HandleFunc("GET /api/workflows/{id}", s.getWorkflow)
	s.mux.HandleFunc("DELETE /api/workflows/{id}", s.deleteWorkflow)
	s.mux.HandleFunc("GET /api/requests", s.listRequests)
	s.mux.HandleFunc("POST /api/requests", s.submitRequest)
	s.mux.HandleFunc("GET /api/requests/{id}", s.getRequest)
	s.mux.HandleFunc("POST /api/requests/{id}/approve", s.approve)
	s.mux.HandleFunc("POST /api/requests/{id}/reject", s.reject)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard); s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
s.mux.HandleFunc("GET /api/tier",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]any{"tier":s.limits.Tier,"upgrade_url":"https://stockyard.dev/escrow/"})})
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, code int, msg string) { writeJSON(w, code, map[string]string{"error": msg}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", http.StatusFound) }
func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"workflows": orEmpty(s.db.ListWorkflows())}) }
func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf store.Workflow; json.NewDecoder(r.Body).Decode(&wf)
	if wf.Name == "" { writeErr(w, 400, "name required"); return }
	s.db.CreateWorkflow(&wf); writeJSON(w, 201, s.db.GetWorkflow(wf.ID))
}
func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) { wf := s.db.GetWorkflow(r.PathValue("id")); if wf == nil { writeErr(w, 404, "not found"); return }; writeJSON(w, 200, wf) }
func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) { s.db.DeleteWorkflow(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }
func (s *Server) listRequests(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"requests": orEmpty(s.db.ListRequests(r.URL.Query().Get("workflow_id"), r.URL.Query().Get("status")))}) }
func (s *Server) submitRequest(w http.ResponseWriter, r *http.Request) {
	var req store.Request; json.NewDecoder(r.Body).Decode(&req)
	if req.Title == "" || req.WorkflowID == "" { writeErr(w, 400, "title and workflow_id required"); return }
	s.db.SubmitRequest(&req); writeJSON(w, 201, s.db.GetRequest(req.ID))
}
func (s *Server) getRequest(w http.ResponseWriter, r *http.Request) { req := s.db.GetRequest(r.PathValue("id")); if req == nil { writeErr(w, 404, "not found"); return }; writeJSON(w, 200, req) }
func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	var d struct{ Approver string `json:"approver"`; Comment string `json:"comment"` }; json.NewDecoder(r.Body).Decode(&d)
	if err := s.db.Decide(r.PathValue("id"), d.Approver, "approved", d.Comment); err != nil { writeErr(w, 400, err.Error()); return }
	writeJSON(w, 200, s.db.GetRequest(r.PathValue("id")))
}
func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	var d struct{ Approver string `json:"approver"`; Comment string `json:"comment"` }; json.NewDecoder(r.Body).Decode(&d)
	if err := s.db.Decide(r.PathValue("id"), d.Approver, "rejected", d.Comment); err != nil { writeErr(w, 400, err.Error()); return }
	writeJSON(w, 200, s.db.GetRequest(r.PathValue("id")))
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) { st := s.db.Stats(); writeJSON(w, 200, map[string]any{"status":"ok","service":"escrow","pending":st.Pending}) }
func orEmpty[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }
