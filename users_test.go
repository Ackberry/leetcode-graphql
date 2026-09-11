package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestLeetcodeUserExistsReturnsFalse(t *testing.T) {
	withFakeLeetcode(t, "getUser", map[string]any{"username": "missing-user"},
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
	withFakeLeetcode(t, "getUserProfile", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	_, err := leetcodeUserProfile(context.Background(), "missing-user")
	if !errors.Is(err, errUserNotFound) {
		t.Fatalf("expected errUserNotFound, got %v", err)
	}
}

func TestLeetcodeUserStatsUserNotFound(t *testing.T) {
	withFakeLeetcode(t, "getUserStats", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	_, err := leetcodeUserStats(context.Background(), "missing-user")
	if !errors.Is(err, errUserNotFound) {
		t.Fatalf("expected errUserNotFound, got %v", err)
	}
}

func TestLeetcodeUserSubmissionsHappyPath(t *testing.T) {
	withFakeLeetcode(t, "getRecentSubmissions", map[string]any{
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
	withFakeLeetcode(t, "getUser", map[string]any{"username": "jimmytrivedi"},
		`{"data":{"matchedUser":{"username":"jimmytrivedi"}}}`)

	exists, err := leetcodeUserExists(context.Background(), "jimmytrivedi")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !exists {
		t.Fatalf("expected user to exist")
	}
}
