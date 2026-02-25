package omf

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Client struct {
	URL      string
	Username string
	Password string
}

func NewClient(url, username, password string) *Client {
	return &Client{
		URL:      url,
		Username: username,
		Password: password,
	}
}

func (c *Client) SetupOMF() {
	// A. Create the Type (The template)
	typeMsg := []interface{}{map[string]interface{}{
		"id": "OleumSensorType", "type": "object", "classification": "dynamic",
		"properties": map[string]interface{}{
			"timestamp": map[string]string{"type": "string", "format": "date-time", "isindex": "true"},
			"value":     map[string]string{"type": "number", "format": "float32"},
		},
	}}
	c.SendToPI("type", typeMsg)

	// B. Create the Container (The actual Tag)
	containerMsg := []interface{}{map[string]string{
		"id": "OGB_10T_THP_PSI", "typeid": "OleumSensorType",
	}}
	c.SendToPI("container", containerMsg)
}

func (c *Client) SendOMFData(containerID string, val float32) error {
	dataMsg := []interface{}{map[string]interface{}{
		"containerid": containerID,
		"values":      []interface{}{map[string]interface{}{"timestamp": time.Now().UTC().Format(time.RFC3339), "value": val}},
	}}
	c.SendToPI("data", dataMsg)
	return nil
}

func (c *Client) SendToPI(msgType string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ JSON Marshal Error: %v", err)
		return err
	}

	req, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("❌ Request Creation Error: %v", err)
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("messagetype", msgType)
	req.Header.Set("omfversion", "1.1")
	req.Header.Set("action", "create")
	req.Header.Set("messageformat", "json")


	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ PI Push Error: %v", err)
		return err
	}
	defer resp.Body.Close()
	print(resp.StatusCode)
	if resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("⚠️ PI Server Response (%s): %d - %s\n", msgType, resp.StatusCode, string(body))
	} else {
		fmt.Printf("✅ Successfully pushed to PI Server (%s)\n", msgType)
	}
	return nil
}
