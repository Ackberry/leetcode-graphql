package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogRequestsPreservesHandlerResponse(t *testing.T) {
	var logs bytes.Buffer
	handler := logRequests(&logs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "created")
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/exists", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("expected readable body, got %v", err)
	}
	if string(body) != "created" {
		t.Fatalf("expected body %q, got %q", "created", string(body))
	}

	logLine := logs.String()
	if !strings.Contains(logLine, "GET /users/jimmytrivedi/exists 201") {
		t.Fatalf("expected request log with method, path, and status, got %q", logLine)
	}
}
