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

// change 1
// change 2
// change 3
// change 4
// change 5
//kdfjskdjf
