package main

import (
    "flag"
	"fmt"
	"log"
	"net/http"
    "time"

	"github.com/skratchdot/open-golang/open"
)

const (
	// PORT = "8081"
	HTML = `<html>
  <head>
    <title>Hello</title>
  </head>
  <body>
    <h1>Hello!</h1>
  </body>
</html>
`
)

var(
    port string
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML)
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "your browser doesn't support server-sent events", 0)
        return
    }

    // Send a comment every second to prevent connection timeout.
    for {
        _, err := fmt.Fprint(w, ": ping")
        if err != nil {
            log.Fatal("client is gone, shutting down")
            return
        }
        flusher.Flush()
        time.Sleep(time.Second)
    }
}

func main() {
    flag.StringVar(&port, "port", "8080", "open browser port")
    flag.Parse()
	http.HandleFunc("/", helloHandler)
	listen := make(chan bool)
	go func() {
		<-listen
		open.Run("http://localhost:" + port + "/")
		log.Println("browser start")
	}()
	listen <- true
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

