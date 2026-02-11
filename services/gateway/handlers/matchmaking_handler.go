package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func MatchmakingHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Proxying matchmaking request: %s %s", r.Method, r.URL.Path)

	host := os.Getenv("MATCHMAKING_HOST")
	if host == "" {
		host = "matchmaking"
	}
	port := os.Getenv("MATCHMAKING_PORT")
	if port == "" {
		port = "8081"
	}

	target := fmt.Sprintf("http://%s:%s", host, port)
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Printf("Error parsing target URL: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
	}

	proxy.ServeHTTP(w, r)
}
