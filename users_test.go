package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLeetcodeUserExistsReturnsTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected content type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body graphQLRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			t.Fatalf("expected valid GraphQL request body, got %v", err)
		}

		if body.Variables["username"] != "jimmytrivedi" {
			t.Fatalf("expected username variable %q, got %v", "jimmytrivedi", body.Variables["username"])
		}

		writeJSON(w, http.StatusOK, graphQLResponse{
			Data: struct {
				MatchedUser *struct {
					Username string `json:"username"`
				} `json:"matchedUser"`
			}{
				MatchedUser: &struct {
					Username string `json:"username"`
				}{
					Username: "jimmytrivedi",
				},
			},
		})
	}))
	defer server.Close()

	oldLeetcode := leetcode
	leetcode = server.URL
	defer func() {
		leetcode = oldLeetcode
	}()

	exists, err := leetcodeUserExists(context.Background(), "jimmytrivedi")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !exists {
		t.Fatalf("expected user to exist")
	}
}
