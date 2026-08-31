package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// constants
var leetcode = "https://leetcode.com/graphql"
var errUserNotFound = errors.New("user not found")

// GraphQL request struct
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// GraphQL response struct
type graphQLResponse struct {
	Data struct {
		MatchedUser *struct {
			Username string `json:"username"`
		} `json:"matchedUser"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// main
func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /users/{username}/exists", handleUserExists)
	mux.HandleFunc("GET /users/{username}/profile", handleUserProfile)
	mux.HandleFunc("GET /users/{username}/stats", handleUserStats)
	mux.HandleFunc("GET /users/{username}/submissions", handleUserSubmissions)
	fmt.Println("starting server on 8080")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users//exists" || r.URL.Path == "/users//profile" || r.URL.Path == "/users//stats" || r.URL.Path == "/users//submissions" {
			writeJSONError(w, http.StatusBadRequest, "username is required")
			return
		}
		mux.ServeHTTP(w, r)
	})

	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		fmt.Println("server error: ", err)
	}
}

type userExistsResponse struct {
	Username string `json:"username"`
	Exists   bool   `json:"exists"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}

func checkLeetcodeStatus(statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("leetcode returned status %d", statusCode)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(data)
}

func postGraphQL(query string, variables map[string]any, result any) error {
	body := graphQLRequest{
		Query:     query,
		Variables: variables,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Post(
		leetcode,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkLeetcodeStatus(resp.StatusCode); err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		return err
	}
	return nil
}

func parseLimit(r *http.Request, defaultLimit int, maxLimit int) (int, error) {
	parsedLimit := defaultLimit
	var err error
	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		parsedLimit, err = strconv.Atoi(limitParam)
		if err != nil || parsedLimit <= 0 || parsedLimit > maxLimit {
			return 0, fmt.Errorf("limit must be an integer between %d and %d", 1, maxLimit)
		}
	}
	return parsedLimit, nil

}

// handlers
func handleUserExists(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	exists, err := leetcodeUserExists(username)
	if err != nil {
		fmt.Printf("exists error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}

	writeJSON(w, http.StatusOK, userExistsResponse{
		Username: username,
		Exists:   exists,
	})
}

type userProfileResponse struct {
	Username    string `json:"username"`
	GithubURL   string `json:"githubUrl"`
	TwitterURL  string `json:"twitterUrl"`
	LinkedinURL string `json:"linkedinUrl"`
	Profile     struct {
		RealName    string   `json:"realName"`
		AboutMe     string   `json:"aboutMe"`
		UserAvatar  string   `json:"userAvatar"`
		CountryName string   `json:"countryName"`
		Company     string   `json:"company"`
		School      string   `json:"school"`
		Websites    []string `json:"websites"`
		SkillTags   []string `json:"skillTags"`
		Ranking     int      `json:"ranking"`
		Reputation  int      `json:"reputation"`
		StarRating  float64  `json:"starRating"`
	} `json:"profile"`
}

type graphQLUserProfileResponse struct {
	Data struct {
		MatchedUser *userProfileResponse `json:"matchedUser"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	profile, err := leetcodeUserProfile(username)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			writeJSONError(w, http.StatusNotFound, "user not found")
			return
		}
		fmt.Printf("profile error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func leetcodeUserProfile(username string) (userProfileResponse, error) {
	query := `
		query getUserProfile($username: String!){
			matchedUser(username: $username) {
				username
				githubUrl
				twitterUrl
				linkedinUrl
				profile {
					realName
					aboutMe
					userAvatar
					countryName
					company
					school
					websites
					skillTags
					ranking
					reputation
					starRating
				}
			}
		}
	`
	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"username": username,
		},
	}
	var result graphQLUserProfileResponse
	err := postGraphQL(body.Query, body.Variables, &result)
	if err != nil {
		return userProfileResponse{}, err
	}

	if result.Data.MatchedUser == nil {
		return userProfileResponse{}, errUserNotFound
	}

	if len(result.Errors) > 0 {
		return userProfileResponse{}, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message)
	}
	return *result.Data.MatchedUser, nil
}

// leetcode helpers
func leetcodeUserExists(username string) (bool, error) {

	query := `
		query getUser($username: String!) {
		matchedUser(username: $username) {
		username
		}
	}
	`
	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"username": username,
		},
	}

	var result graphQLResponse
	err := postGraphQL(body.Query, body.Variables, &result)
	if err != nil {
		return false, err
	}

	if result.Data.MatchedUser == nil {
		return false, nil
	}

	if len(result.Errors) > 0 {
		return false, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message) // first error only for simplicity
	}
	return true, nil
}

func handleUserStats(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}
	stats, err := leetcodeUserStats(username)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			writeJSONError(w, http.StatusNotFound, "user not found")
			return
		}
		fmt.Printf("stats error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

type submissionStat struct {
	Difficulty  string `json:"difficulty"`
	Count       int    `json:"count"`
	Submissions int    `json:"submissions"`
}

type userStatsResponse struct {
	Username    string `json:"username"`
	SubmitStats struct {
		AcceptedSubmissions []submissionStat `json:"acSubmissionNum"`
		TotalSubmissions    []submissionStat `json:"totalSubmissionNum"`
	} `json:"submitStats"`
}

type graphQLUserStatsResponse struct {
	Data struct {
		MatchedUser *userStatsResponse `json:"matchedUser"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func leetcodeUserStats(username string) (userStatsResponse, error) {
	query := `
		query getUserStats($username: String!) {
			matchedUser(username: $username) {
				username
				submitStats {
					acSubmissionNum {
					difficulty
					count
					submissions
					}
					totalSubmissionNum {
					difficulty
					count
					submissions
					}
				}
			}
		}
	`
	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"username": username,
		},
	}
	var result graphQLUserStatsResponse
	err := postGraphQL(body.Query, body.Variables, &result)
	if err != nil {
		return userStatsResponse{}, err
	}

	if result.Data.MatchedUser == nil {
		return userStatsResponse{}, errUserNotFound
	}
	if len(result.Errors) > 0 {
		return userStatsResponse{}, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message)
	}
	return *result.Data.MatchedUser, nil
}

func handleUserSubmissions(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	parsedLimit := 10
	var err error
	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		parsedLimit, err = strconv.Atoi(limitParam)
		if err != nil || parsedLimit <= 0 || parsedLimit > 20 {
			writeJSONError(w, http.StatusBadRequest, "limit must be an integer between 1 and 20")
			return
		}
	}
	limit, err := parseLimit(r, 10, 20)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ok, err := leetcodeUserExists(username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	if !ok {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	recentSubmissions, err := leetcodeUserSubmissions(username, limit)
	if err != nil {
		fmt.Printf("recent submissions error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	writeJSON(w, http.StatusOK, recentSubmissions)
}

type userSubmission struct {
	Title         string `json:"title"`
	TitleSlug     string `json:"titleSlug"`
	Timestamp     string `json:"timestamp"`
	StatusDisplay string `json:"statusDisplay"`
	Lang          string `json:"lang"`
}

type userSubmissionsResponse struct {
	Username          string           `json:"username"`
	RecentSubmissions []userSubmission `json:"recentSubmissions"`
}

type graphQLUserSubmissionsResponse struct {
	Data struct {
		RecentSubmissionList []userSubmission `json:"recentSubmissionList"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func leetcodeUserSubmissions(username string, limit int) (userSubmissionsResponse, error) {
	query := `
		query getRecentSubmissions($username: String!, $limit: Int!) {
			recentSubmissionList(username: $username, limit: $limit) {
				title
				titleSlug
				timestamp
				statusDisplay
				lang
			}
		}
	`
	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"username": username,
			"limit":    limit,
		},
	}

	var result graphQLUserSubmissionsResponse
	err := postGraphQL(body.Query, body.Variables, &result)
	if err != nil {
		return userSubmissionsResponse{}, err
	}
	if len(result.Errors) > 0 {
		return userSubmissionsResponse{}, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message)
	}
	return userSubmissionsResponse{
		Username:          username,
		RecentSubmissions: result.Data.RecentSubmissionList,
	}, nil
}
