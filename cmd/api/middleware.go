package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

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

	type client struct {
		ratelimiter *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu sync.Mutex
		clients = make(map[string]*client)
	)

	go func ()  {
		for {
			time.Sleep(time.Minute)

			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > (3 * time.Minute) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request)  {

		if (app.config.limiter.enabled) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)

			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
	
			mu.Lock()
	
			if _, exists := clients[ip]; !exists {
				clients[ip] = &client{
					ratelimiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}
	
			clients[ip].lastSeen = time.Now()
	
			if !clients[ip].ratelimiter.Allow() {
				mu.Unlock()
				app.ratelimitExceededResponse(w, r)
				return
			}
	
			mu.Unlock()
	
		}
		
		next.ServeHTTP(w, r)
	})
}