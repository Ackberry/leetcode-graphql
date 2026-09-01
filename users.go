package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// constants
var errUserNotFound = errors.New("user not found")

type userExistsResponse struct {
	Username string `json:"username"`
	Exists   bool   `json:"exists"`
}

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
