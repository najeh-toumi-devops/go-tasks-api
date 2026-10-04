package task

import "context"

// Store est une INTERFACE : elle décrit ce qu'on sait faire, pas comment.
// En Go, un type implémente une interface implicitement, juste en ayant
// les bonnes méthodes. Ça nous permet d'avoir 2 implémentations :
// MemoryStore (tests, démarrage rapide) et PostgresStore (production).
type Store interface {
	List(ctx context.Context) ([]Task, error)
	Get(ctx context.Context, id int64) (Task, error)
	Create(ctx context.Context, in Input) (Task, error)
	Update(ctx context.Context, id int64, in Input) (Task, error)
	Delete(ctx context.Context, id int64) error
}
