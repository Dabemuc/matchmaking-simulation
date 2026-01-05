package pool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func getGatewayURL() string {
	hostname := os.Getenv("GATEWAY_HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("http://%s:8080", hostname)
}

func Login(username, password string) error {
	url := getGatewayURL() + "/login"
	requestBody, err := json.Marshal(map[string]string{
		"username": username,
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
