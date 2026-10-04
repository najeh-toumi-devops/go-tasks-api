package task

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer monte le Handler avec le MemoryStore : pas besoin de BDD
// pour tester. C'est tout l'intérêt de l'interface Store.
func newTestServer() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(NewMemoryStore()).Register(mux)
	return mux
}

func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCreateAndGet(t *testing.T) {
	mux := newTestServer()

	rec := do(mux, "POST", "/tasks", `{"title":"Apprendre Go"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", rec.Code)
	}
	var created Task
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Title != "Apprendre Go" {
		t.Errorf("unexpected task: %+v", created)
	}

	rec = do(mux, "GET", "/tasks/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200", rec.Code)
	}
}

// Test "table-driven" : l'idiome de test le plus courant en Go.
func TestErrors(t *testing.T) {
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"titre vide", "POST", "/tasks", `{"title":""}`, 422},
		{"json invalide", "POST", "/tasks", `{oops`, 400},
		{"champ inconnu", "POST", "/tasks", `{"titel":"x"}`, 400},
		{"id invalide", "GET", "/tasks/abc", "", 400},
		{"introuvable", "GET", "/tasks/42", "", 404},
		{"delete introuvable", "DELETE", "/tasks/42", "", 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(newTestServer(), tc.method, tc.path, tc.body)
			if rec.Code != tc.want {
				t.Errorf("got %d, want %d (body: %s)", rec.Code, tc.want, rec.Body)
			}
		})
	}
}

func TestUpdateAndDelete(t *testing.T) {
	mux := newTestServer()
	do(mux, "POST", "/tasks", `{"title":"A"}`)

	rec := do(mux, "PUT", "/tasks/1", `{"title":"A modifiée","done":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: got %d", rec.Code)
	}
	var updated Task
	_ = json.NewDecoder(rec.Body).Decode(&updated)
	if !updated.Done {
		t.Error("expected done=true")
	}

	if rec := do(mux, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d", rec.Code)
	}
	if rec := do(mux, "GET", "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("after delete: got %d, want 404", rec.Code)
	}
}
