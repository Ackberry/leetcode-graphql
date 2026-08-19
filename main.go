package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var leetcode = "https://leetcode.com/graphql"

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
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

func main() {

	http.HandleFunc("GET /users/{username}/exists", handleUserExists)

	fmt.Println("starting server on 8080")

	err := http.ListenAndServe(":8080", nil)
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

func handleUserExists(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	exists, err := userExists(username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(errorResponse{
			Error: "failed to connect to leetcode. try again",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userExistsResponse{
		Username: username,
		Exists:   exists,
	})
}

func userExists(username string) (bool, error) {
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

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("leetcode returned status %d", resp.StatusCode)
	}

	var result graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false, err
	}
	if len(result.Errors) > 0 {
		return false, fmt.Errorf("leetcode graphql error: %s", result.Errors[0].Message) // first error only for simplicity
	}
	return result.Data.MatchedUser != nil, nil
}
