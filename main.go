package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

var leetcode = "https://leetcode.com/graphql"

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data struct {
		MatchedUser *struct {
			Username string `json:"username`
		} `json:"matchedUser"`
	} `json:"data"`
}

func main() {

	http.HandleFunc("GET /users/{username}/exists", handleUserExists)

	fmt.Println("starting server on 8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("server error: ", err)
	}
}

func handleUserExists(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	exists, err := userExists(username)
	if err != nil {
		http.Error(w, "failed to connect to leetcode. try again", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s exists: %t", username, exists)
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

	resp, err := http.Post(
		leetcode,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	defer resp.Body.Close()
	if err != nil {
		return false, err
	}

	var result graphQLResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false, err
	}
	return result.Data.MatchedUser != nil, nil
}
