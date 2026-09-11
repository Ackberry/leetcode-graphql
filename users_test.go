package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestLeetcodeUserExistsReturnsFalse(t *testing.T) {
	fakeGraphQL(t, "getUser", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	exists, err := leetcodeUserExists(context.Background(), "missing-user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatal("expected user not to exist")
	}
}

func TestLeetcodeUserProfileUserNotFound(t *testing.T) {
	fakeGraphQL(t, "getUserProfile", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	_, err := leetcodeUserProfile(context.Background(), "missing-user")
	if !errors.Is(err, errUserNotFound) {
		t.Fatalf("expected errUserNotFound, got %v", err)
	}
}

func TestLeetcodeUserStatsUserNotFound(t *testing.T) {
	fakeGraphQL(t, "getUserStats", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	_, err := leetcodeUserStats(context.Background(), "missing-user")
	if !errors.Is(err, errUserNotFound) {
		t.Fatalf("expected errUserNotFound, got %v", err)
	}
}

func TestLeetcodeUserSubmissionsHappyPath(t *testing.T) {
	fakeGraphQL(t, "getRecentSubmissions", map[string]any{
		"username": "jimmytrivedi",
		"limit":    float64(2), // JSON numbers decode into float64 in map[string]any.
	}, `{"data":{"recentSubmissionList":[
		{"title":"Two Sum","titleSlug":"two-sum","timestamp":"1725000000","statusDisplay":"Accepted","lang":"golang"},
		{"title":"Add Two Numbers","titleSlug":"add-two-numbers","timestamp":"1724990000","statusDisplay":"Wrong Answer","lang":"python3"}
	]}}`)

	got, err := leetcodeUserSubmissions(context.Background(), "jimmytrivedi", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := userSubmissionsResponse{
		Username: "jimmytrivedi",
		RecentSubmissions: []userSubmission{
			{Title: "Two Sum", TitleSlug: "two-sum", Timestamp: "1725000000", StatusDisplay: "Accepted", Lang: "golang"},
			{Title: "Add Two Numbers", TitleSlug: "add-two-numbers", Timestamp: "1724990000", StatusDisplay: "Wrong Answer", Lang: "python3"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected submissions %+v, got %+v", want, got)
	}
}

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
