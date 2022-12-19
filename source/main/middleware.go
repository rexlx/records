package main

import (
	"fmt"
	"net/http"
)

func (app *Application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := app.validateKey(r)
		if err != nil {
			data := jsonResponse{
				Error:   true,
				Message: fmt.Sprintf("bad key, %v", err),
			}
			_ = app.writeJSON(w, http.StatusUnauthorized, data)
			return
		}
		next.ServeHTTP(w, r)
	})
}
