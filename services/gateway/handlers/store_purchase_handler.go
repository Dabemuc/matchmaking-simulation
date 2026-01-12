package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

/*
 This function forwards store purchase requests to the store service or mocks a basic response
*/

func StorePurchaseHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Printf("received request on %s from %s", r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Id       string `json:"id"`
		Offer_Id string `json:"offer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("received store purchase request for user: %s", req.Id)

	w.WriteHeader(http.StatusOK)
}
