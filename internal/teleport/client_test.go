package teleport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestDeviceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deviceToken/mlRequestClientAccess" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-devicetoken") != "AB123" {
			t.Errorf("unexpected x-devicetoken: %s", r.Header.Get("x-devicetoken"))
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["client_hint"] != "my-uuid" {
			t.Errorf("unexpected client_hint: %s", body["client_hint"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"client_id": "device-token-123",
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	token, err := client.RequestDeviceToken("my-uuid", "AB123")
	if err != nil {
		t.Fatalf("RequestDeviceToken() error: %v", err)
	}
	if token != "device-token-123" {
		t.Errorf("token = %q, want %q", token, "device-token-123")
	}
}

func TestRequestDeviceTokenFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "invalid pin",
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	_, err := client.RequestDeviceToken("my-uuid", "BAD")
	if err == nil {
		t.Error("expected error for failed request")
	}
}

func TestGetICEConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deviceToken/mlIceConfig" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"servers": []interface{}{
				map[string]interface{}{"urls": "stun:stun.example.com:3478"},
			},
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	servers, err := client.GetICEConfig("token123")
	if err != nil {
		t.Fatalf("GetICEConfig() error: %v", err)
	}

	// servers is json.RawMessage (a byte slice), so len() returns byte
	// count, not array length. Unmarshal into a slice to check the count.
	var parsed []json.RawMessage
	if err := json.Unmarshal(servers, &parsed); err != nil {
		t.Fatalf("failed to unmarshal servers: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("got %d servers, want 1", len(parsed))
	}
}

func TestSignalingConnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deviceToken/mlClientConnect" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"answer":  "v=0\r\ns=-\r\n",
		})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	answer, err := client.SignalingConnect("offer-sdp", json.RawMessage(`[{"urls":"stun:stun.example.com"}]`), "token123")
	if err != nil {
		t.Fatalf("SignalingConnect() error: %v", err)
	}
	if answer != "v=0\r\ns=-\r\n" {
		t.Errorf("answer = %q, unexpected", answer)
	}
}
