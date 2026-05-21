package main

import (
	"ForumJS/internal/repository"
	"log"
	"net/http"
	"os"
)

func main() {
	db, err := repository.InitDB("./migrations")
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Forum is running"))
	})

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
