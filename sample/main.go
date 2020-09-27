package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Print("starting server...")
	http.HandleFunc("/", handler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, _ *http.Request) {
	name := os.Getenv("NAME")
	if name == "" {
		name = "0.^.0"
	}
	_, err := fmt.Fprintf(w, "Hello %s!\n", name)
	if err != nil {
		log.Println("error writing to response stream")
	}
}
