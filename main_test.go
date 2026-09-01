package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmptyUsernameReturnsBadRequest(t *testing.T) {
	handler := newServerHandler()

	req := httptest.NewRequest(http.MethodGet, "/users//exists", nil)
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
}
