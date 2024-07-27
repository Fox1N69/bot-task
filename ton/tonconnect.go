package ton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TonConnectClient struct {
	baseURL string
	apiKey  string
}

func NewTonConnectClient(baseURL, apiKey string) *TonConnectClient {
	return &TonConnectClient{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (c *TonConnectClient) ConnectWallet(userID int) error {
	url := fmt.Sprintf("%s/connect_wallet", c.baseURL)
	fmt.Printf("Request URL: %s\n", url) // Логирование для отладки

	payload := map[string]int{
		"user_id": userID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// Process response
	return nil
}
