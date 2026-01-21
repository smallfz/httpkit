package main

import (
	"github.com/smallfz/httpkit/kit"
	"net/http"
)

func home() string {
	return "Hello!"
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", kit.F(home))

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
