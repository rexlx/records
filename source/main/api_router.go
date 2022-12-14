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
		mux.Get("/runtime/stats", app.ListAllCounters)
		mux.Post("/runtime/kill", app.KillService)
		mux.Post("/runtime/start", app.StartService)

		mux.Post("/service/store", app.GetStore)
		mux.Post("/service/runtime", app.GetRuntime)
		mux.Post("/service/errors", app.GetErrorsById)
	})
	// might need static files later
	// fserver := http.FileServer(http.Dir("./static/"))
	// mux.Handle("/static/*", http.StripPrefix("/static", fserver))
	return mux
}
