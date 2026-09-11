package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// Tests using the fake must remain serial because leetcode is a shared endpoint.
func withFakeLeetcode(t *testing.T, operation string, variables map[string]any, response string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected content type application/json, got %s", got)
		}
		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("expected valid GraphQL request body, got %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if !strings.Contains(body.Query, "query "+operation+"(") {
			t.Errorf("expected GraphQL operation %q, got %q", operation, body.Query)
		}
		if !reflect.DeepEqual(body.Variables, variables) {
			t.Errorf("expected variables %v, got %v", variables, body.Variables)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	oldLeetcode := leetcode
	leetcode = server.URL
	t.Cleanup(func() {
		server.Close()
		leetcode = oldLeetcode
	})
}
