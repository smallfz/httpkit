package main

import (
    "net/http"
    "github.com/smallfz/httpkit/kit"
)

func home() string {
    return "Hello!"
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", kit.F(home))

    server := http.Server{
        Addr: ":8080",
        Handler: mux,
    }
    server.ListenAndServe()
}
