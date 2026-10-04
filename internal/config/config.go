// Package config lit la configuration depuis les variables d'environnement.
// Bonne pratique "12-factor app" : pas de config en dur dans le code.
package config

import "os"

type Config struct {
	Port        string // port HTTP d'écoute
	DatabaseURL string // vide => on utilise le stockage en mémoire
}

// Load construit la Config. getenv renvoie une valeur par défaut si la variable est absente.
func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
