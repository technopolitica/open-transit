package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Env struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.AllowContentType("application/vnd.mds+json"))
	router.Use(middleware.Heartbeat("/health"))

	env := Env{db}

	vehiclesRouter := NewVehiclesRouter(&env)
	router.Mount("/vehicles", vehiclesRouter)

	return router
}
