package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/skratchdot/open-golang/open"
)

const (
	PORT = "8081"
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

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, HTML)
}

func main() {
	http.HandleFunc("/", helloHandler)
	listen := make(chan bool)
	go func() {
		<-listen
		open.Run("http://localhost:" + PORT + "/")
		log.Println("browser start")
	}()
	listen <- true
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
