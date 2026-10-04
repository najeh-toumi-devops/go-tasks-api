package task

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore garde les tâches dans une map.
// Le serveur HTTP traite chaque requête dans sa propre goroutine,
// donc on protège la map avec un mutex (sinon : data race).
type MemoryStore struct {
	mu     sync.RWMutex
	tasks  map[int64]Task
	nextID int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[int64]Task), nextID: 1}
}

func (s *MemoryStore) List(_ context.Context) ([]Task, error) {
	s.mu.RLock()         // verrou en lecture (plusieurs lecteurs possibles)
	defer s.mu.RUnlock() // defer : exécuté à la sortie de la fonction

	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	// Une map n'a pas d'ordre garanti : on trie par ID.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryStore) Get(_ context.Context, id int64) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id] // idiome "comma ok"
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) Create(_ context.Context, in Input) (Task, error) {
	s.mu.Lock() // verrou exclusif en écriture
	defer s.mu.Unlock()
	t := Task{
		ID:        s.nextID,
		Title:     strings.TrimSpace(in.Title),
		Done:      in.Done,
		CreatedAt: time.Now().UTC(),
	}
	s.tasks[t.ID] = t
	s.nextID++
	return t, nil
}

func (s *MemoryStore) Update(_ context.Context, id int64, in Input) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	t.Title = strings.TrimSpace(in.Title)
	t.Done = in.Done
	s.tasks[id] = t
	return t, nil
}

func (s *MemoryStore) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
