package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// constants
var leetcode = "https://leetcode.com/graphql"
var errUserNotFound = errors.New("user not found")
var errProblemNotFound = errors.New("problem not found")
var leetcodeClient = &http.Client{
	Timeout: 5 * time.Second,
}

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
	mux.HandleFunc("GET /problems/{slug}", handleProblem)
	fmt.Println("starting server on 8080")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users//exists" || r.URL.Path == "/users//profile" || r.URL.Path == "/users//stats" || r.URL.Path == "/users//submissions" {
			writeJSONError(w, http.StatusBadRequest, "username is required")
			return
		}
		mux.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	err := server.ListenAndServe()
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

func postGraphQL(ctx context.Context, query string, variables map[string]any, result any) error {
	body := graphQLRequest{
		Query:     query,
		Variables: variables,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		leetcode,
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := leetcodeClient.Do(req)
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

func htmlToText(htmlString string) string {
	doc, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return htmlString
	}

	var parts []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				parts = append(parts, text)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return strings.Join(parts, " ")
}

// handlers
func handleUserExists(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	exists, err := leetcodeUserExists(r.Context(), username)
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

	profile, err := leetcodeUserProfile(r.Context(), username)
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

func leetcodeUserProfile(ctx context.Context, username string) (userProfileResponse, error) {
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
	err := postGraphQL(ctx, body.Query, body.Variables, &result)
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
func leetcodeUserExists(ctx context.Context, username string) (bool, error) {

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
	err := postGraphQL(ctx, body.Query, body.Variables, &result)
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
	stats, err := leetcodeUserStats(r.Context(), username)
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

func leetcodeUserStats(ctx context.Context, username string) (userStatsResponse, error) {
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
	err := postGraphQL(ctx, body.Query, body.Variables, &result)
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

	limit, err := parseLimit(r, 10, 20)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ok, err := leetcodeUserExists(r.Context(), username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	if !ok {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}

	recentSubmissions, err := leetcodeUserSubmissions(r.Context(), username, limit)
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

func leetcodeUserSubmissions(ctx context.Context, username string, limit int) (userSubmissionsResponse, error) {
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
	err := postGraphQL(ctx, body.Query, body.Variables, &result)
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

func handleProblem(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		writeJSONError(w, http.StatusBadRequest, "problem slug is required")
		return
	}

	problem, err := leetcodeProblem(r.Context(), slug)
	if err != nil {
		if errors.Is(err, errProblemNotFound) {
			writeJSONError(w, http.StatusNotFound, "problem not found")
			return
		}
		fmt.Printf("problem error: %s\n", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to connect to leetcode. try again")
		return
	}
	writeJSON(w, http.StatusOK, problem)
}

type topicTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type problemResponse struct {
	QuestionFrontendID string     `json:"questionFrontendId"`
	Title              string     `json:"title"`
	TitleSlug          string     `json:"titleSlug"`
	Difficulty         string     `json:"difficulty"`
	IsPaidOnly         bool       `json:"isPaidOnly"`
	ACRate             float64    `json:"acRate"`
	Likes              int        `json:"likes"`
	Dislikes           int        `json:"dislikes"`
	Content            string     `json:"content"`
	TopicTags          []topicTag `json:"topicTags"`
}

type graphQLProblemResponse struct {
	Data struct {
		Question *problemResponse `json:"question"`
	} `json:"data"`

	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func leetcodeProblem(ctx context.Context, slug string) (problemResponse, error) {
	query := `query getProblem($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionFrontendId
			title
			titleSlug
			difficulty
			isPaidOnly
			acRate
			likes
			dislikes
			content
			topicTags {
				name
				slug
			}
		}
	}`

	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"titleSlug": slug,
		},
	}
	var result graphQLProblemResponse

	err := postGraphQL(ctx, body.Query, body.Variables, &result)
	if err != nil {
		return problemResponse{}, err
	}
	if result.Data.Question == nil {
		return problemResponse{}, errProblemNotFound
	}
	if len(result.Errors) > 0 {
		return problemResponse{}, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message)
	}
	problem := *result.Data.Question
	problem.Content = htmlToText(problem.Content)

	return problem, nil
}
