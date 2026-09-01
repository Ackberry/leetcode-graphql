package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
