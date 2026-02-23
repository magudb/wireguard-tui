package teleport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://client.amplifi.com"
	userAgent      = "AmpliFiTeleport/7 CFNetwork/1220.1 Darwin/20.3.0"
)

// Client communicates with the Amplifi Teleport API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Client pointing at the production Amplifi API.
func NewClient() *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) post(path, token string, body any) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encoding request: %w", err)
		}
	}

	url := c.BaseURL + path
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-devicetoken", token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return data, nil
}

// RequestDeviceToken exchanges a PIN for a persistent device token.
func (c *Client) RequestDeviceToken(clientHint, pin string) (string, error) {
	data, err := c.post("/api/deviceToken/mlRequestClientAccess", pin,
		map[string]string{"client_hint": clientHint})
	if err != nil {
		return "", err
	}

	var resp struct {
		Success  bool   `json:"success"`
		Error    string `json:"error"`
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("device token request failed: %s", resp.Error)
	}
	return resp.ClientID, nil
}

// GetICEConfig fetches ICE server configuration.
func (c *Client) GetICEConfig(token string) (json.RawMessage, error) {
	data, err := c.post("/api/deviceToken/mlIceConfig", token, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Success bool            `json:"success"`
		Error   string          `json:"error"`
		Servers json.RawMessage `json:"servers"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing ICE config: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("ICE config request failed: %s", resp.Error)
	}
	return resp.Servers, nil
}

// SignalingConnect sends an SDP offer and returns the SDP answer.
func (c *Client) SignalingConnect(offer string, iceServers json.RawMessage, token string) (string, error) {
	data, err := c.post("/api/deviceToken/mlClientConnect", token, map[string]any{
		"iceServers": iceServers,
		"offer":      offer,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Answer  string `json:"answer"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parsing signaling response: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("signaling connect failed: %s", resp.Error)
	}
	return resp.Answer, nil
}
