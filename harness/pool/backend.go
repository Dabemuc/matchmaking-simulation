package pool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
)

func getGatewayURL() string {
	hostname := os.Getenv("GATEWAY_HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("http://%s:8080", hostname)
}

func Login(id int, password string) error {
	url := getGatewayURL() + "/login"
	requestBody, err := json.Marshal(map[string]string{
		"id":       strconv.Itoa(id),
		"password": password,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func FetchStore() error {
	url := getGatewayURL() + "/store/offers"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching store failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func StorePurchase(id int) error {
	url := getGatewayURL() + "/store/purchase"

	requestBody, err := json.Marshal(map[string]string{
		"id":       strconv.Itoa(id),
		"offer_id": strconv.Itoa(int(rand.Uint())),
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("store purchase failed with status code: %d", resp.StatusCode)
	}

	return nil
}
