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
		// here we can use middleware for anything in the `app` route
		mux.Post("/runtime", app.ListServices)
	})
	// static files
	fserver := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fserver))
	return mux
}
