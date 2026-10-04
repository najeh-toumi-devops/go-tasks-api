// Package task contient tout le "domaine" Task : modèle, stockage, handlers HTTP.
package task

import (
	"errors"
	"strings"
	"time"
)

// Task est notre entité. Les "struct tags" `json:"..."` indiquent
// le nom du champ quand on encode/décode en JSON.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Input est ce que le client envoie pour créer/modifier une tâche.
// On sépare Input et Task : le client ne choisit ni l'ID ni la date.
type Input struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Erreurs "sentinelles" : on peut les comparer avec errors.Is(err, ErrNotFound).
var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("title is required (max 200 chars)")
)

// Validate est une méthode sur Input (receiver "in Input").
func (in Input) Validate() error {
	t := strings.TrimSpace(in.Title)
	if t == "" || len(t) > 200 {
		return ErrInvalidTitle
	}
	return nil
}
