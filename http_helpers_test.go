package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseLimitUsingDefaultWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/submissions", nil)

	limit, err := parseLimit(req, 10, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if limit != 10 {
		t.Fatalf("expected limit 10, got %d", limit)
	}
}

func TestParseLimitUsesQueryLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/submissions?limit=5", nil)

	limit, err := parseLimit(req, 10, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if limit != 5 {
		t.Fatalf("expected limit 5, got %d", limit)
	}
}

func TestParseLimitRejectsInvalidLimits(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{{
		name: "non integer",
		url:  "/users/jimmytrivedi/submissions?limit=abc",
	},
		{
			name: "zero",
			url:  "/users/jimmytrivedi/submissions?limit=0",
		},
		{
			name: "negative",
			url:  "/users/jimmytrivedi/submissions?limit=-12",
		},
		{
			name: "above max",
			url:  "/users/jimmytrivedi/submissions?limit=33",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			_, err := parseLimit(req, 10, 20)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})

	}
}

func TestWriteJSONError(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeJSONError(recorder, http.StatusBadRequest, "limit must be valid")

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected content type application/json, got %s", resp.Header.Get("Content-Type"))
	}

	var body errorResponse

	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}
	if body.Error != "limit must be valid" {
		t.Fatalf("expected error message %q, got %q", "limit must be valid", body.Error)
	}
}
