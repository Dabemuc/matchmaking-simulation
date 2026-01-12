package handlers

import (
	"log"
	"net/http"
)

/*
 This function forwards store offers requests to the store service or mocks a basic response
*/

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
