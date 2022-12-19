package main

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func (app *Application) apiRoutes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)

	mux.Route("/app", func(mux chi.Router) {
		mux.Use(app.authenticate)
		// here we can use middleware for anything in the `app` route
		mux.Get("/runtime", app.ListServices)
		mux.Get("/runtime/loaded", app.ListLoaded)
		mux.Post("/kill", app.KillService)
	})
	// might need static files later
	fserver := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fserver))
	return mux
}
