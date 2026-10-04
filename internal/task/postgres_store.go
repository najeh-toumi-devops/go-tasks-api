package task

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// PostgresStore implémente Store avec PostgreSQL via database/sql
// (l'interface SQL standard de Go). Le driver pgx est enregistré dans main.go.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) List(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, done, created_at FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // toujours fermer les rows !

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (Task, error) {
	var t Task
	// $1 = paramètre : protège contre l'injection SQL. Ne JAMAIS concaténer.
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound // on traduit l'erreur SQL en erreur du domaine
	}
	return t, err
}

func (s *PostgresStore) Create(ctx context.Context, in Input) (Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO tasks (title, done) VALUES ($1, $2)
		 RETURNING id, title, done, created_at`,
		strings.TrimSpace(in.Title), in.Done,
	).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	return t, err
}

func (s *PostgresStore) Update(ctx context.Context, id int64, in Input) (Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx,
		`UPDATE tasks SET title = $1, done = $2 WHERE id = $3
		 RETURNING id, title, done, created_at`,
		strings.TrimSpace(in.Title), in.Done, id,
	).Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
