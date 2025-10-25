package config

import (
	"log"
	"os"
)

type Config struct {
	PGURL       string
	StanCluster string
	StanClient  string
	StanURL     string
	StanSubject string
	HTTPAddr    string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	cfg := Config{
		PGURL:       getenv("PG_URL", "postgres://demo:demo@localhost:5432/ordersdb"),
		StanCluster: getenv("STAN_CLUSTER", "orders-cluster"),
		StanClient:  getenv("STAN_CLIENT", "orders-service"),
		StanURL:     getenv("STAN_URL", "nats://localhost:4222"),
		StanSubject: getenv("STAN_SUBJECT", "orders"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
	}
	log.Printf("cfg loaded: http=%s, subject=%s", cfg.HTTPAddr, cfg.StanSubject)
	return cfg
}
