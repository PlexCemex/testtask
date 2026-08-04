package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"taskservice/internal/auth"
	"taskservice/internal/middleware"
)

type Router struct {
	JWT       *auth.JWTManager
	Auth      *AuthHandler
	Teams     *TeamHandler
	Tasks     *TaskHandler
	RateLimit *middleware.RateLimiter
}

func (rt *Router) Build() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Metrics)
	r.Use(rt.RateLimit.Middleware)

	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/register", rt.Auth.Register)
		api.Post("/login", rt.Auth.Login)

		api.Group(func(priv chi.Router) {
			priv.Use(middleware.Auth(rt.JWT))

			priv.Post("/teams", rt.Teams.Create)
			priv.Get("/teams", rt.Teams.List)
			priv.Post("/teams/{id}/invite", rt.Teams.Invite)
			priv.Get("/teams/stats", rt.Teams.Stats)
			priv.Get("/teams/{id}/top-creators", rt.Teams.TopCreators)
			priv.Get("/teams/{id}/orphan-assignees", rt.Teams.OrphanAssignees)

			priv.Post("/tasks", rt.Tasks.Create)
			priv.Get("/tasks", rt.Tasks.List)
			priv.Put("/tasks/{id}", rt.Tasks.Update)
			priv.Get("/tasks/{id}/history", rt.Tasks.History)
		})
	})

	return r
}
