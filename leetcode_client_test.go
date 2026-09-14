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
	withFakeLeetcodeCalls(t, fakeGraphQLCall{
		operation: operation,
		variables: variables,
		response:  response,
	})
}

type fakeGraphQLCall struct {
	operation string
	variables map[string]any
	response  string
	status    int
}

func withFakeLeetcodeCalls(t *testing.T, calls ...fakeGraphQLCall) {
	t.Helper()
	callIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callIndex >= len(calls) {
			t.Errorf("received unexpected GraphQL request")
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		call := calls[callIndex]
		callIndex++

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
		if !strings.Contains(body.Query, "query "+call.operation+"(") {
			t.Errorf("expected GraphQL operation %q, got %q", call.operation, body.Query)
		}
		if !reflect.DeepEqual(body.Variables, call.variables) {
			t.Errorf("expected variables %v, got %v", call.variables, body.Variables)
		}
		w.Header().Set("Content-Type", "application/json")
		status := call.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, call.response)
	}))
	oldLeetcode := leetcode
	leetcode = server.URL
	t.Cleanup(func() {
		server.Close()
		leetcode = oldLeetcode
		if callIndex != len(calls) {
			t.Errorf("expected %d GraphQL calls, got %d", len(calls), callIndex)
		}
	})
}
