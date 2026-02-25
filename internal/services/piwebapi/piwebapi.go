package piwebapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	// "time"
)

type ServiceClient struct {
	BaseURL  string
	Username string
	Password string
}

type StreamValue struct {
	Timestamp string      `json:"timestamp"`
	Value     interface{} `json:"value"`
}

func NewServiceClient(baseURL, username, password string) *ServiceClient {
	return &ServiceClient{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
	}
}

func (c *ServiceClient) PushValue(webID string, value interface{}) error {
	url := fmt.Sprintf("%s/streams/%s/value", c.BaseURL, webID)

	// payload := StreamValue{
	// 	Timestamp: time.Now().UTC().Format(time.RFC3339),
	// 	Value:     value,
	// }
payload := map[string]interface{}{
		"Value": value,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ JSON Marshal Error: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("❌ Request Creation Error: %v", err)
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "go-client")
	httpClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ PI Web API Push Error: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️ PI Web API Error: %d - %s\n", resp.StatusCode, string(body))
	} else {
		fmt.Printf("✅ Successfully pushed to PI Web API (WebID: %s)\n", webID)
	}
	return nil
}
