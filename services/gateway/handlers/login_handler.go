package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

/*
 This function forwards login requests to the authentication service or mocks a basic response
*/

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Printf("received request on %s from %s", r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Id       string `json:"id"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("received login request for user: %s", req.Id)

	w.WriteHeader(http.StatusOK)
}
