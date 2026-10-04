// Package migrations embarque le SQL DANS le binaire grâce à //go:embed.
// Résultat : un seul fichier exécutable à déployer, rien d'autre.
package migrations

import _ "embed"

//go:embed 001_create_tasks.sql
var Schema string
