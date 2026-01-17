
# httpkit

A super simple but handy web framework for net/http of golang.

```go
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
```

More to see in folder examples.