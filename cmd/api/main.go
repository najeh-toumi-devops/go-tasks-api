// Point d'entrée du programme. Un exécutable Go = package main + func main().
package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // import "_" : enregistre le driver "pgx"

	"github.com/najeh-toumi-devops/go-tasks-api/internal/config"
	"github.com/najeh-toumi-devops/go-tasks-api/internal/task"
	"github.com/najeh-toumi-devops/go-tasks-api/migrations"
)

func main() {
	// Logs structurés en JSON (pratique pour Loki, ELK, etc.)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// run contient la vraie logique : renvoyer une erreur est plus propre
// et plus testable que d'appeler log.Fatal partout.
func run() error {
	cfg := config.Load()

	// 1. Choix du stockage selon la config
	store, cleanup, err := newStore(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	// 2. Routes
	mux := http.NewServeMux()
	task.NewHandler(store).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok")) // sonde liveness/readiness pour Kubernetes
	})

	// 3. Serveur HTTP avec timeouts (ne jamais utiliser les valeurs par défaut en prod)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 4. On lance le serveur dans une goroutine...
	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 5. ...et on attend soit une erreur, soit un signal d'arrêt (Ctrl+C, SIGTERM de k8s)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down...")
	}

	// 6. Graceful shutdown : on laisse 10s aux requêtes en cours pour finir
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newStore renvoie l'interface task.Store : main ne sait pas (et n'a pas
// besoin de savoir) quelle implémentation est derrière.
func newStore(cfg config.Config) (task.Store, func(), error) {
	if cfg.DatabaseURL == "" {
		slog.Warn("DATABASE_URL not set, using in-memory store (data lost on restart)")
		return task.NewMemoryStore(), func() {}, nil
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, err
	}
	if _, err := db.ExecContext(ctx, migrations.Schema); err != nil {
		db.Close()
		return nil, nil, err
	}
	slog.Info("connected to PostgreSQL")
	return task.NewPostgresStore(db), func() { db.Close() }, nil
}

// logging est un MIDDLEWARE : une fonction qui enveloppe un handler
// pour ajouter un comportement (ici : logguer chaque requête).
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// statusRecorder "embarque" http.ResponseWriter (embedding) et surcharge
// WriteHeader pour mémoriser le code de statut.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
