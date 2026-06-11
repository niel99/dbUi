package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/niel/dbui/server/internal/adapter"
	cassandraAdapter "github.com/niel/dbui/server/internal/adapter/cassandra"
	mongoAdapter "github.com/niel/dbui/server/internal/adapter/mongodb"
	pgAdapter "github.com/niel/dbui/server/internal/adapter/postgres"
	"github.com/niel/dbui/server/internal/connection"
	"github.com/niel/dbui/server/internal/handler"
	"github.com/niel/dbui/server/internal/models"
	"github.com/niel/dbui/server/internal/registry"
)

func main() {
	// Create plugin registry and register adapters
	reg := registry.New()

	reg.Register(models.DBTypePostgres, func() adapter.DatabaseAdapter {
		return pgAdapter.New()
	})
	reg.Register(models.DBTypeMongoDB, func() adapter.DatabaseAdapter {
		return mongoAdapter.New()
	})
	reg.Register(models.DBTypeCassandra, func() adapter.DatabaseAdapter {
		return cassandraAdapter.New(models.DBTypeCassandra)
	})
	reg.Register(models.DBTypeScyllaDB, func() adapter.DatabaseAdapter {
		return cassandraAdapter.New(models.DBTypeScyllaDB)
	})

	// Create connection manager and handler
	mgr := connection.NewManager(reg)
	h := handler.New(mgr)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Mount("/", h.Routes())

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("dbUI server starting on :%s\n", port)
	fmt.Printf("Registered adapters: %v\n", reg.ListTypes())
	log.Fatal(http.ListenAndServe(":"+port, r))
}
