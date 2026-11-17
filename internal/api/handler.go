package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler serves the plant-list read routes.
type Handler struct {
	store Store
}

// NewMux registers GET /health, /sites, /sites/{id}, and /reactors.
func NewMux(store Store) http.Handler {
	h := &Handler{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /sites", h.listSites)
	mux.HandleFunc("GET /sites/{id}", h.getSite)
	mux.HandleFunc("GET /reactors", h.listReactors)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.store.ListSites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, sites)
}

func (h *Handler) getSite(w http.ResponseWriter, r *http.Request) {
	detail, err := h.store.SiteByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) listReactors(w http.ResponseWriter, r *http.Request) {
	reactors, err := h.store.ListReactors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reactors)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}
