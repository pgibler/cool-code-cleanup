package main

import "net/http"

func main() {
	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {})
	http.HandleFunc("/payment/secure", func(w http.ResponseWriter, r *http.Request) {})
	http.ListenAndServe(":8000", nil)
}
