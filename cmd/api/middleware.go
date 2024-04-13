package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)


func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		defer func ()  {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) ratelimit(next http.Handler) http.Handler {
	limit := rate.NewLimiter(2, 4)

	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {
		if !limit.Allow() {
			app.ratelimitExceededResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}