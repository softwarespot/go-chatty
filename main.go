package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/socket.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/socket.js")
	})
	http.HandleFunc("/queryParams.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/queryParams.js")
	})
	http.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/app.js")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})

	cs := NewChatServer()
	http.Handle("/chat", cs)

	http.ListenAndServe(":10000", nil)
}
