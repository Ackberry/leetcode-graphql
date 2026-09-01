package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{username}/exists", handleUserExists)
	mux.HandleFunc("GET /users/{username}/profile", handleUserProfile)
	mux.HandleFunc("GET /users/{username}/stats", handleUserStats)
	mux.HandleFunc("GET /users/{username}/submissions", handleUserSubmissions)
	mux.HandleFunc("GET /problems/{slug}", handleProblem)
	fmt.Println("starting server on 8080")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users//exists" || r.URL.Path == "/users//profile" || r.URL.Path == "/users//stats" || r.URL.Path == "/users//submissions" {
			writeJSONError(w, http.StatusBadRequest, "username is required")
			return
		}
		mux.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("server error: ", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
