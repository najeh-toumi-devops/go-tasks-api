# go-tasks-api

API REST de gestion de tâches développée en Go. Le projet met en pratique la bibliothèque standard (`net/http`, `encoding/json`, `database/sql`) et utilise PostgreSQL via le driver `pgx`.

L'API propose les opérations CRUD, deux stockages interchangeables (mémoire et PostgreSQL), des tests HTTP sans base de données et une image Docker minimale. Le code peut également servir de support pour découvrir Go.

**Dépôt GitHub :** [najeh-toumi-devops/go-tasks-api](https://github.com/najeh-toumi-devops/go-tasks-api)

Pour récupérer le projet :

```bash
git clone https://github.com/najeh-toumi-devops/go-tasks-api.git
cd go-tasks-api
```

## Sommaire

- [Fonctionnalités](#fonctionnalités)
- [Prérequis](#prérequis)
- [Démarrage rapide](#démarrage-rapide)
- [Configuration](#configuration)
- [API HTTP](#api-http)
- [Docker Compose](#docker-compose)
- [Tests et qualité](#tests-et-qualité)
- [Architecture](#architecture)
- [Concepts Go étudiés](#concepts-go-étudiés)
- [Pistes d'amélioration](#pistes-damélioration)

## Fonctionnalités

- Créer, consulter, modifier et supprimer des tâches.
- Choisir le stockage mémoire ou PostgreSQL avec `DATABASE_URL`.
- Créer le schéma PostgreSQL au démarrage de l'application.
- Valider les entrées JSON et limiter les corps de requête à 1 Mo.
- Journaliser les requêtes en JSON et arrêter proprement le serveur.
- Vérifier que l'API répond avec `GET /healthz`.
- Exécuter les tests avec le détecteur de data races de Go.

## Prérequis

- Go 1.22 ou une version ultérieure.
- Docker avec Docker Compose v2 pour le démarrage conteneurisé (facultatif).
- `curl` ou un client HTTP. [`requests.http`](requests.http) fonctionne avec l'extension REST Client de VS Code.

## Démarrage rapide

### Sans base de données

Lorsque `DATABASE_URL` n'est pas défini, l'application utilise le stockage mémoire. Les données sont perdues à l'arrêt du processus.

```bash
go mod tidy
go run ./cmd/api
```

L'API écoute sur `http://localhost:8080`. Pour lancer les tests :

```bash
go test -race ./...
```

### Vérifier l'API

Créer une tâche :

```bash
curl -i -X POST http://localhost:8080/tasks \
   -H "Content-Type: application/json" \
   -d '{"title":"Apprendre Go"}'
```

Lister les tâches :

```bash
curl http://localhost:8080/tasks
```

## Configuration

| Variable | Requise | Valeur par défaut | Description |
|----------|---------|-------------------|-------------|
| `PORT` | Non | `8080` | Port d'écoute HTTP. |
| `DATABASE_URL` | Non | Aucune | URL PostgreSQL. Si elle est absente ou vide, le stockage mémoire est utilisé. |

Exemple d'URL pour une base locale :

```text
postgres://tasks:tasks@localhost:5432/tasks?sslmode=disable
```

Avec PostgreSQL, l'application vérifie la connexion au démarrage et exécute le schéma intégré au binaire. Elle ne démarre pas si la connexion ou l'initialisation échoue.

## API HTTP

Les routes utilisent le routeur standard de Go 1.22. Les identifiants `{id}` doivent être des entiers positifs.

| Méthode | Route | Description | Réponse en cas de succès |
|---------|-------|-------------|--------------------------|
| `GET` | `/healthz` | Vérifier que le serveur répond. | `200 OK`, texte `ok` |
| `GET` | `/tasks` | Lister les tâches. | `200 OK`, tableau JSON |
| `POST` | `/tasks` | Créer une tâche. | `201 Created`, tâche JSON et en-tête `Location` |
| `GET` | `/tasks/{id}` | Lire une tâche. | `200 OK`, tâche JSON |
| `PUT` | `/tasks/{id}` | Remplacer le titre et l'état d'une tâche. | `200 OK`, tâche JSON |
| `DELETE` | `/tasks/{id}` | Supprimer une tâche. | `204 No Content` |

Le corps des requêtes `POST` et `PUT` est un objet JSON. `title` est obligatoire après suppression des espaces et ne doit pas dépasser 200 octets. `done` est un booléen qui vaut `false` s'il est omis.

```json
{
   "title": "Apprendre Go",
   "done": false
}
```

Exemple de réponse :

```json
{
   "id": 1,
   "title": "Apprendre Go",
   "done": false,
   "created_at": "2026-10-04T12:00:00Z"
}
```

Les erreurs sont renvoyées en JSON sous la forme `{"error":"..."}`.

| Code | Signification |
|------|---------------|
| `400 Bad Request` | JSON invalide, champ inconnu ou identifiant incorrect. |
| `404 Not Found` | Tâche inexistante. |
| `422 Unprocessable Entity` | Titre vide ou de plus de 200 octets. |
| `500 Internal Server Error` | Erreur interne ; les détails sont conservés dans les journaux du serveur. |

Exemples des autres opérations :

```bash
curl http://localhost:8080/tasks/1

curl -i -X PUT http://localhost:8080/tasks/1 \
   -H "Content-Type: application/json" \
   -d '{"title":"Apprendre Go et PostgreSQL","done":true}'

curl -i -X DELETE http://localhost:8080/tasks/1
```

## Docker Compose

Docker Compose démarre PostgreSQL et l'API. Les données PostgreSQL sont conservées dans le volume nommé `pgdata`.

```bash
docker compose up --build
```

L'API est disponible sur `http://localhost:8080`. Pour démarrer en arrière-plan et consulter les journaux :

```bash
docker compose up --build -d
docker compose logs -f api
```

Arrêter les conteneurs sans supprimer les données :

```bash
docker compose down
```

Attention : `make down` exécute `docker compose down -v` et supprime aussi le volume `pgdata` et les données de la base.

L'image applicative est construite en plusieurs étapes et utilise une image Distroless non-root à l'exécution.

## Tests et qualité

```bash
go test -race ./...  # tests avec détection des data races
go vet ./...         # vérifications statiques
gofmt -w .           # formatage Go
```

Avec Make installé, les cibles disponibles comprennent `make test`, `make vet`, `make fmt`, `make build`, `make run`, `make run-db`, `make up` et `make down`. `make cover` produit un rapport de couverture HTML.

Les tests HTTP utilisent `httptest` et le stockage mémoire : ils ne nécessitent ni PostgreSQL ni Docker.

## Architecture

```text
.
├── cmd/api/main.go              # configuration, serveur HTTP, arrêt propre
├── internal/
│   ├── config/config.go         # variables d'environnement
│   └── task/
│       ├── model.go             # modèles, validation et erreurs métier
│       ├── store.go             # interface du stockage
│       ├── memory_store.go      # stockage mémoire protégé par mutex
│       ├── postgres_store.go    # stockage PostgreSQL avec database/sql
│       ├── handler.go           # routes et réponses HTTP/JSON
│       └── handler_test.go      # tests des handlers
├── migrations/
│   ├── 001_create_tasks.sql     # schéma SQL
│   └── migrations.go            # schéma intégré au binaire
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── requests.http                # requêtes manuelles REST Client
```

Les handlers dépendent de l'interface `Store`, pas d'une implémentation particulière. Le choix entre mémoire et PostgreSQL est effectué au démarrage dans `cmd/api`. En Go, les packages d'`internal/` ne peuvent être importés que depuis ce module et ses sous-répertoires.

## Concepts Go étudiés

| Fichier | Concepts principaux |
|---------|---------------------|
| `internal/task/model.go` | Structures, tags JSON, méthodes et erreurs sentinelles. |
| `internal/config/config.go` | Fonctions, retours multiples et variables d'environnement. |
| `internal/task/store.go` | Interfaces et `context.Context`. |
| `internal/task/memory_store.go` | Maps, slices, pointeurs et `sync.RWMutex`. |
| `internal/task/handler.go` | `net/http`, routage Go 1.22, JSON et gestion des erreurs. |
| `internal/task/handler_test.go` | `testing`, `httptest` et tests table-driven. |
| `internal/task/postgres_store.go` | `database/sql`, requêtes paramétrées et erreurs SQL. |
| `cmd/api/main.go` | Goroutines, channels, middleware, signaux et arrêt propre. |

## Pistes d'amélioration

- Ajouter filtrage et pagination à `GET /tasks`.
- Ajouter une route `PATCH` pour les mises à jour partielles.
- Compléter les tests du stockage mémoire et ajouter des tests d'intégration PostgreSQL.
- Ajouter authentification, métriques et manifests de déploiement Kubernetes.

## Ressources

- [Documentation officielle Go](https://go.dev/doc/)
- [A Tour of Go](https://go.dev/tour/)
- [Go by Example](https://gobyexample.com/)
- [Documentation PostgreSQL](https://www.postgresql.org/docs/)

---

Développé par **Najeh TOUMI**.
