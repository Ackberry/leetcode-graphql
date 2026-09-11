package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunServerShutdown(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "finishes active request"
		if force {
			name = "closes request after deadline"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			started := make(chan struct{})
			release := make(chan struct{}, 1)
			defer close(release)
			requestCanceled := make(chan struct{})
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-release:
					io.WriteString(w, "finished")
				case <-r.Context().Done():
					close(requestCanceled)
				}
			})}
			defer server.Close()
			shutdownStarted := make(chan struct{})
			server.RegisterOnShutdown(func() { close(shutdownStarted) })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			timeout := 5 * time.Second
			if force {
				timeout = 50 * time.Millisecond
			}
			done := make(chan error, 1)
			go func() { done <- runServer(ctx, server, listener, timeout) }()
			response := make(chan error, 1)
			go func() {
				client := &http.Client{Timeout: 5 * time.Second}
				defer client.CloseIdleConnections()
				resp, err := client.Get("http://" + listener.Addr().String())
				if err == nil {
					defer resp.Body.Close()
					var body []byte
					body, err = io.ReadAll(resp.Body)
					if err == nil && string(body) != "finished" {
						err = errors.New("incomplete response")
					}
				}
				response <- err
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not start")
			}
			cancel()
			select {
			case <-shutdownStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown did not start")
			}
			if !force {
				select {
				case err := <-done:
					t.Fatalf("server exited before request completed: %v", err)
				default:
				}
				release <- struct{}{}
			}
			select {
			case err := <-done:
				if force && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected shutdown deadline, got %v", err)
				}
				if !force && err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("server did not stop")
			}
			select {
			case err := <-response:
				if force && err == nil {
					t.Fatal("expected active connection to be closed")
				}
				if !force && err != nil {
					t.Fatalf("request did not finish cleanly: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("client did not finish")
			}
			if force {
				select {
				case <-requestCanceled:
				case <-time.After(5 * time.Second):
					t.Fatal("request context was not canceled")
				}
			}
		})
	}
}

func TestRunServerReturnsServeError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = runServer(ctx, &http.Server{}, listener, time.Second)
	if !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected listener error, got %v", err)
	}
}

func TestEmptyUsernameReturnsBadRequest(t *testing.T) {
	handler := newServerHandler()

	tests := []struct {
		name string
		path string
	}{
		{name: "exists", path: "/users//exists"},
		{name: "profile", path: "/users//profile"},
		{name: "stats", path: "/users//stats"},
		{name: "submissions", path: "/users//submissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			resp := recorder.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", resp.StatusCode)
			}

			var body errorResponse
			err := json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("expected valid JSON body, got %v", err)
			}

			if body.Error != "username is required" {
				t.Fatalf("expected error %q, got %q", "username is required", body.Error)
			}
		})
	}
}
