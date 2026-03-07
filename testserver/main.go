package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Server starting version 2...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello World")
	})

	http.ListenAndServe(":8080", nil)
}
