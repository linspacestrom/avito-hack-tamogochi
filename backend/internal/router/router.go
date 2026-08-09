package router

import (
	"log/slog"
	"net/http"

	_ "github.com/NBx03/avito-hack-tamagotchi/backend/docs"
	"github.com/NBx03/avito-hack-tamagotchi/backend/internal/handler"
	appmiddleware "github.com/NBx03/avito-hack-tamagotchi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func New(
	h *handler.Handler,
	auth appmiddleware.Authenticator,
	log *slog.Logger,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.RequestLogger(log))
	r.Use(chimiddleware.Recoverer)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
		r.With(appmiddleware.RequireAuth(auth, log)).Get("/me", h.Me)
	})

	r.Route("/api/rewards", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(auth, log))
		r.Post("/issue", h.IssueReward)
		r.Get("/me", h.ListMyRewards)
		r.Get("/daily-status", h.DailyStatus)
		r.Post("/daily-claim", h.DailyClaim)
	})

	r.Route("/api/tasks", func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(auth, log))
		r.Get("/me", h.ListMyTasks)
		r.Post("/{code}/claim", h.ClaimTask)
	})

	return r
}
