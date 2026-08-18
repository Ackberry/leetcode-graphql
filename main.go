package main

import (
	"fmt"
	"net/http"
)

func main() {

	http.HandleFunc("/users/", handleUserExists)

	fmt.Println("starting server on 8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("server error: ", err)
	}
}

func handleUserExists(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "exists route works")
}
