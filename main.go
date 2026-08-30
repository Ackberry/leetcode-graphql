package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	fmt.Println("starting server on 8080")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "//") {
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

// handlers
func handleUserExists(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}

	exists, err := leetcodeUserExists(username)
	if err != nil {
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
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return userProfileResponse{}, err
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
		return userProfileResponse{}, err
	}
	defer resp.Body.Close()

	if err := checkLeetcodeStatus(resp.StatusCode); err != nil {
		return userProfileResponse{}, err
	}

	var result graphQLUserProfileResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
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
	if username == "" {

	}

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

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return false, err
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
		return false, err
	}
	defer resp.Body.Close()

	if err := checkLeetcodeStatus(resp.StatusCode); err != nil {
		return false, err
	}

	var result graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
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
