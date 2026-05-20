package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Forum is running"))
	})

	log.Println("Server starting on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
