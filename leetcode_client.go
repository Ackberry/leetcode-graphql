package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// constants
var leetcode = "https://leetcode.com/graphql"
var leetcodeClient = &http.Client{
	Timeout: 5 * time.Second,
}

func checkLeetcodeStatus(statusCode int) error {
	if statusCode != http.StatusOK {
		return fmt.Errorf("leetcode returned status %d", statusCode)
	}
	return nil
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
