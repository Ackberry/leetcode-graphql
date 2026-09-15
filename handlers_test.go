package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func performRequest(handler http.Handler, path string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	return recorder.Result()
}

func decodeJSONBody(t *testing.T, resp *http.Response, body any) {
	t.Helper()
	err := json.NewDecoder(resp.Body).Decode(body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}
}

func assertErrorResponse(t *testing.T, resp *http.Response, status int, message string) {
	t.Helper()
	defer resp.Body.Close()

	if resp.StatusCode != status {
		t.Fatalf("expected status %d, got %d", status, resp.StatusCode)
	}

	var body errorResponse
	decodeJSONBody(t, resp, &body)
	if body.Error != message {
		t.Fatalf("expected error %q, got %q", message, body.Error)
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
			resp := performRequest(handler, tt.path)
			assertErrorResponse(t, resp, http.StatusBadRequest, "username is required")
		})
	}
}

func TestUserExistsHandlerReturnsTrue(t *testing.T) {
	withFakeLeetcode(t, "getUser", map[string]any{"username": "jimmytrivedi"},
		`{"data":{"matchedUser":{"username":"jimmytrivedi"}}}`)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/exists", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body userExistsResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	want := userExistsResponse{
		Username: "jimmytrivedi",
		Exists:   true,
	}
	if body != want {
		t.Fatalf("expected response %+v, got %+v", want, body)
	}
}

func TestUserExistsHandlerReturnsFalse(t *testing.T) {
	withFakeLeetcode(t, "getUser", map[string]any{"username": "missing-user"},
		`{"data":{"matchedUser":null}}`)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/missing-user/exists", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body userExistsResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	want := userExistsResponse{
		Username: "missing-user",
		Exists:   false,
	}
	if body != want {
		t.Fatalf("expected response %+v, got %+v", want, body)
	}
}

func TestUserExistsHandlerReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	oldLeetcode := leetcode
	leetcode = server.URL
	defer func() {
		leetcode = oldLeetcode
	}()

	handler := newServerHandler()
	resp := performRequest(handler, "/users/jimmytrivedi/exists")
	assertErrorResponse(t, resp, http.StatusInternalServerError, "failed to connect to leetcode. try again")
}

func TestUserProfileHandlerReturnsProfile(t *testing.T) {
	withFakeLeetcode(t, "getUserProfile", map[string]any{"username": "jimmytrivedi"}, `{
		"data": {
			"matchedUser": {
				"username": "jimmytrivedi",
				"githubUrl": "https://github.com/jimmytrivedi",
				"twitterUrl": "https://x.com/MrJimmyTrivedi",
				"linkedinUrl": "https://linkedin.com/in/jimmytrivedi",
				"profile": {
					"realName": "Jimmy Trivedi",
					"aboutMe": "Software engineer",
					"userAvatar": "https://assets.leetcode.com/users/jimmytrivedi/avatar.png",
					"countryName": "India",
					"company": "Zivame",
					"school": "Gujarat University",
					"websites": ["https://jimmytrivedi.in"],
					"skillTags": ["android", "kotlin", "java"],
					"ranking": 1016466,
					"reputation": 12,
					"starRating": 2
				}
			}
		}
	}`)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/profile", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body userProfileResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	if body.Username != "jimmytrivedi" {
		t.Fatalf("expected username %q, got %q", "jimmytrivedi", body.Username)
	}
	if body.GithubURL != "https://github.com/jimmytrivedi" {
		t.Fatalf("expected github URL to round-trip, got %q", body.GithubURL)
	}
	if body.Profile.RealName != "Jimmy Trivedi" {
		t.Fatalf("expected real name %q, got %q", "Jimmy Trivedi", body.Profile.RealName)
	}
	if body.Profile.Ranking != 1016466 {
		t.Fatalf("expected ranking 1016466, got %d", body.Profile.Ranking)
	}
	if len(body.Profile.SkillTags) != 3 || body.Profile.SkillTags[0] != "android" {
		t.Fatalf("expected skill tags to round-trip, got %+v", body.Profile.SkillTags)
	}
}

func TestUserStatsHandlerReturnsStats(t *testing.T) {
	withFakeLeetcode(t, "getUserStats", map[string]any{"username": "jimmytrivedi"}, `{
		"data": {
			"matchedUser": {
				"username": "jimmytrivedi",
				"submitStats": {
					"acSubmissionNum": [
						{"difficulty": "All", "count": 30, "submissions": 40},
						{"difficulty": "Easy", "count": 20, "submissions": 25}
					],
					"totalSubmissionNum": [
						{"difficulty": "All", "count": 35, "submissions": 65},
						{"difficulty": "Easy", "count": 22, "submissions": 35}
					]
				}
			}
		}
	}`)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/stats", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body userStatsResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	if body.Username != "jimmytrivedi" {
		t.Fatalf("expected username %q, got %q", "jimmytrivedi", body.Username)
	}
	if len(body.SubmitStats.AcceptedSubmissions) != 2 {
		t.Fatalf("expected 2 accepted submission buckets, got %+v", body.SubmitStats.AcceptedSubmissions)
	}
	if body.SubmitStats.AcceptedSubmissions[0] != (submissionStat{Difficulty: "All", Count: 30, Submissions: 40}) {
		t.Fatalf("expected accepted stats to round-trip, got %+v", body.SubmitStats.AcceptedSubmissions[0])
	}
	if body.SubmitStats.TotalSubmissions[1] != (submissionStat{Difficulty: "Easy", Count: 22, Submissions: 35}) {
		t.Fatalf("expected total stats to round-trip, got %+v", body.SubmitStats.TotalSubmissions[1])
	}
}

func TestUserSubmissionsHandlerReturnsSubmissions(t *testing.T) {
	withFakeLeetcodeCalls(t,
		fakeGraphQLCall{
			operation: "getUser",
			variables: map[string]any{"username": "jimmytrivedi"},
			response:  `{"data":{"matchedUser":{"username":"jimmytrivedi"}}}`,
		},
		fakeGraphQLCall{
			operation: "getRecentSubmissions",
			variables: map[string]any{
				"username": "jimmytrivedi",
				"limit":    float64(2),
			},
			response: `{"data":{"recentSubmissionList":[
				{"title":"Two Sum","titleSlug":"two-sum","timestamp":"1725000000","statusDisplay":"Accepted","lang":"golang"},
				{"title":"Add Two Numbers","titleSlug":"add-two-numbers","timestamp":"1724990000","statusDisplay":"Wrong Answer","lang":"python3"}
			]}}`,
		},
	)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/users/jimmytrivedi/submissions?limit=2", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body userSubmissionsResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	want := userSubmissionsResponse{
		Username: "jimmytrivedi",
		RecentSubmissions: []userSubmission{
			{Title: "Two Sum", TitleSlug: "two-sum", Timestamp: "1725000000", StatusDisplay: "Accepted", Lang: "golang"},
			{Title: "Add Two Numbers", TitleSlug: "add-two-numbers", Timestamp: "1724990000", StatusDisplay: "Wrong Answer", Lang: "python3"},
		},
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("expected response %+v, got %+v", want, body)
	}
}

func TestProblemHandlerReturnsProblem(t *testing.T) {
	withFakeLeetcode(t, "getProblem", map[string]any{"titleSlug": "two-sum"}, `{
		"data": {
			"question": {
				"questionFrontendId": "1",
				"title": "Two Sum",
				"titleSlug": "two-sum",
				"difficulty": "Easy",
				"isPaidOnly": false,
				"acRate": 58.09,
				"likes": 69861,
				"dislikes": 2607,
				"content": "<p>You are given an <strong>array</strong>.</p><ul><li><code>2 &lt;= nums.length &lt;= 10<sup>4</sup></code></li></ul>",
				"topicTags": [
					{"name": "Array", "slug": "array"},
					{"name": "Hash Table", "slug": "hash-table"}
				]
			}
		}
	}`)

	handler := newServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/problems/two-sum", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body problemResponse
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}

	if body.TitleSlug != "two-sum" {
		t.Fatalf("expected title slug %q, got %q", "two-sum", body.TitleSlug)
	}
	if body.Content != "You are given an array.\n\n- 2 <= nums.length <= 10^4" {
		t.Fatalf("expected plain text content, got %q", body.Content)
	}
	wantTags := []topicTag{
		{Name: "Array", Slug: "array"},
		{Name: "Hash Table", Slug: "hash-table"},
	}
	if !reflect.DeepEqual(body.TopicTags, wantTags) {
		t.Fatalf("expected topic tags %+v, got %+v", wantTags, body.TopicTags)
	}
}

func TestHandlersReturnNotFound(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		operation string
		variables map[string]any
		response  string
		wantError string
	}{
		{
			name:      "profile missing user",
			path:      "/users/missing-user/profile",
			operation: "getUserProfile",
			variables: map[string]any{"username": "missing-user"},
			response:  `{"data":{"matchedUser":null}}`,
			wantError: "user not found",
		},
		{
			name:      "stats missing user",
			path:      "/users/missing-user/stats",
			operation: "getUserStats",
			variables: map[string]any{"username": "missing-user"},
			response:  `{"data":{"matchedUser":null}}`,
			wantError: "user not found",
		},
		{
			name:      "submissions missing user",
			path:      "/users/missing-user/submissions",
			operation: "getUser",
			variables: map[string]any{"username": "missing-user"},
			response:  `{"data":{"matchedUser":null}}`,
			wantError: "user not found",
		},
		{
			name:      "missing problem",
			path:      "/problems/missing-problem",
			operation: "getProblem",
			variables: map[string]any{"titleSlug": "missing-problem"},
			response:  `{"data":{"question":null}}`,
			wantError: "problem not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFakeLeetcode(t, tt.operation, tt.variables, tt.response)

			handler := newServerHandler()
			resp := performRequest(handler, tt.path)
			assertErrorResponse(t, resp, http.StatusNotFound, tt.wantError)
		})
	}
}

func TestSubmissionsHandlerRejectsInvalidLimit(t *testing.T) {
	handler := newServerHandler()

	tests := []struct {
		name string
		path string
	}{
		{name: "non integer", path: "/users/jimmytrivedi/submissions?limit=abc"},
		{name: "zero", path: "/users/jimmytrivedi/submissions?limit=0"},
		{name: "negative", path: "/users/jimmytrivedi/submissions?limit=-1"},
		{name: "above max", path: "/users/jimmytrivedi/submissions?limit=21"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(handler, tt.path)
			assertErrorResponse(t, resp, http.StatusBadRequest, "limit must be an integer between 1 and 20")
		})
	}
}

func TestHandlersReturnUpstreamError(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		operation string
		variables map[string]any
	}{
		{
			name:      "profile upstream error",
			path:      "/users/jimmytrivedi/profile",
			operation: "getUserProfile",
			variables: map[string]any{"username": "jimmytrivedi"},
		},
		{
			name:      "stats upstream error",
			path:      "/users/jimmytrivedi/stats",
			operation: "getUserStats",
			variables: map[string]any{"username": "jimmytrivedi"},
		},
		{
			name:      "problem upstream error",
			path:      "/problems/two-sum",
			operation: "getProblem",
			variables: map[string]any{"titleSlug": "two-sum"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFakeLeetcodeCalls(t, fakeGraphQLCall{
				operation: tt.operation,
				variables: tt.variables,
				status:    http.StatusInternalServerError,
				response:  `{"error":"upstream unavailable"}`,
			})

			handler := newServerHandler()
			resp := performRequest(handler, tt.path)
			assertErrorResponse(t, resp, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		})
	}
}

func TestSubmissionsHandlerReturnsUpstreamErrorAfterUserCheck(t *testing.T) {
	withFakeLeetcodeCalls(t,
		fakeGraphQLCall{
			operation: "getUser",
			variables: map[string]any{"username": "jimmytrivedi"},
			response:  `{"data":{"matchedUser":{"username":"jimmytrivedi"}}}`,
		},
		fakeGraphQLCall{
			operation: "getRecentSubmissions",
			variables: map[string]any{
				"username": "jimmytrivedi",
				"limit":    float64(10),
			},
			status:   http.StatusInternalServerError,
			response: `{"error":"upstream unavailable"}`,
		},
	)

	handler := newServerHandler()
	resp := performRequest(handler, "/users/jimmytrivedi/submissions")
	assertErrorResponse(t, resp, http.StatusInternalServerError, "failed to connect to leetcode. try again")
}
