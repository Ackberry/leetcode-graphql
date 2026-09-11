package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	fmt.Println("starting server on", port)
	return runServer(ctx, server, listener, 15*time.Second)
}

func runServer(ctx context.Context, server *http.Server, listener net.Listener, timeout time.Duration) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		fmt.Println("shutting down server")
	}

	// Keep active request contexts alive while they finish, even though the
	// signal context has been canceled.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		closeErr := server.Close()
		return errors.Join(fmt.Errorf("graceful shutdown: %w", err), closeErr)
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
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
