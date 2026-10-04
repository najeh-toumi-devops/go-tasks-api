package task

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

// Handler fait le lien HTTP <-> Store. Il dépend de l'interface,
// pas d'une implémentation concrète (= injection de dépendance).
type Handler struct {
	store Store
}

func NewHandler(s Store) *Handler {
	return &Handler{store: s}
}

// Register déclare les routes. Depuis Go 1.22, le routeur standard
// gère la méthode HTTP et les paramètres de chemin ({id}).
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /tasks", h.list)
	mux.HandleFunc("POST /tasks", h.create)
	mux.HandleFunc("GET /tasks/{id}", h.get)
	mux.HandleFunc("PUT /tasks/{id}", h.update)
	mux.HandleFunc("DELETE /tasks/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.store.List(r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	t, err := h.store.Create(r.Context(), in)
	if err != nil {
		h.serverError(w, err)
		return
	}
	w.Header().Set("Location", "/tasks/"+strconv.FormatInt(t.ID, 10))
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	t, err := h.store.Update(r.Context(), id, in)
	if err != nil {
		h.handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.handleErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204 : succès, pas de corps
}

// ---------- helpers ----------

// parseID renvoie 2 valeurs : Go permet les retours multiples.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, bool) {
	var in Input
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // max 1 Mo
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // refuse {"titel": "..."} (faute de frappe)
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return Input{}, false
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return Input{}, false
	}
	return in, true
}

func (h *Handler) handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	h.serverError(w, err)
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)                        // on loggue le détail...
	writeError(w, http.StatusInternalServerError, "internal error") // ...mais on ne l'expose pas
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
