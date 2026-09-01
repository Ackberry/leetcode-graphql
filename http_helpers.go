package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: message,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(data)
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
