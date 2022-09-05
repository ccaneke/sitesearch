package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "logger: ", log.Lshortfile)

	mux := http.NewServeMux()
	handler := handler{logger: logger}

	fileServer := http.FileServer(http.Dir("./ui/static"))

	mux.HandleFunc("/", handler.home)
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	mux.HandleFunc("/search", handler.search)

	logger.Print("Starting server on :4000")
	err := http.ListenAndServe(":4000", mux)
	logger.Fatal(err)
}
