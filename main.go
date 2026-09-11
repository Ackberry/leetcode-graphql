package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	handler := newServerHandler()
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Println("starting server on", port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("server error: ", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func newServerHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{username}/exists", handleUserExists)
	mux.HandleFunc("GET /users/{username}/profile", handleUserProfile)
	mux.HandleFunc("GET /users/{username}/stats", handleUserStats)
	mux.HandleFunc("GET /users/{username}/submissions", handleUserSubmissions)
	mux.HandleFunc("GET /problems/{slug}", handleProblem)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emptyUsernamePaths := map[string]bool{
			"/users//exists":      true,
			"/users//profile":     true,
			"/users//stats":       true,
			"/users//submissions": true,
		}
		if emptyUsernamePaths[r.URL.Path] {
			writeJSONError(w, http.StatusBadRequest, "username is required")
			return
		}
		mux.ServeHTTP(w, r)
	})
}
