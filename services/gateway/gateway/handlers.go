package gateway

import (
	"encoding/json"
	"log"
	"net/http"
)

/*
 These functions forward requests to the appropriate backend services or mock basic responses
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

func StoreOffersHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Printf("received request on %s from %s", r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("received store offers request")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	mockResponse := []byte(`
	{
		"offers": [
			{
				"id": "offer_1",
				"title": "Blue Skin",
				"price": 9.99,
				"currency": "USD"
			},
			{
				"id": "offer_2",
				"title": "Red Skin",
				"price": 5.99,
				"currency": "USD"
			},
			{
				"id": "offer_3",
				"title": "Green Skin",
				"price": 5.99,
				"currency": "USD"
			},
			{
				"id": "offer_4",
				"title": "Yellow Skin",
				"price": 7.99,
				"currency": "USD"
			},
			{
				"id": "offer_5",
				"title": "Purple Skin",
				"price": 15.99,
				"currency": "USD"
			},
			{
				"id": "offer_6",
				"title": "Pink Skin",
				"price": 1.99,
				"currency": "USD"
			}
		]
	}
	`)

	w.Write(mockResponse)
}

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
