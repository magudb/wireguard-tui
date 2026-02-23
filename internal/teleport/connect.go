package teleport

import (
	"fmt"
	"os"
	"time"

	"github.com/mlu/wireguard-tui/internal/wg"
	"github.com/pion/webrtc/v4"
)

const (
	stunServer     = "stun:global.stun.twilio.com:3478"
	devicePlatform = "iOS"
	iceTimeout     = 30 * time.Second
	// CredentialDir is where Teleport tokens and UUIDs are stored.
	CredentialDir = "/etc/wireguard/.teleport"
)

// ConnectResult holds the output of a successful Teleport connection.
type ConnectResult struct {
	ConfigText string
	Name       string
}

// Connect performs the full Amplifi Teleport protocol.
// If pin is non-empty, authenticates first (saves token for future reconnects).
// If pin is empty, uses a previously saved token.
// Returns the WireGuard config text.
func Connect(pin, name string) (*ConnectResult, error) {
	client := NewClient()

	var deviceToken string
	if pin != "" {
		// First-time: exchange PIN for device token
		uuid, err := LoadOrCreateUUID(CredentialDir, name)
		if err != nil {
			return nil, fmt.Errorf("loading UUID: %w", err)
		}

		deviceToken, err = client.RequestDeviceToken(uuid, pin)
		if err != nil {
			return nil, fmt.Errorf("requesting device token: %w", err)
		}

		if err := SaveToken(CredentialDir, name, deviceToken); err != nil {
			return nil, fmt.Errorf("saving device token: %w", err)
		}
	} else {
		// Reconnect: load saved token
		var err error
		deviceToken, err = LoadToken(CredentialDir, name)
		if err != nil {
			return nil, fmt.Errorf("no saved token for %q (use PIN for initial setup): %w", name, err)
		}
	}

	configText, err := connectWithToken(client, deviceToken)
	if err != nil {
		return nil, err
	}

	return &ConnectResult{ConfigText: configText, Name: name}, nil
}

func connectWithToken(client *Client, deviceToken string) (string, error) {
	// Generate WireGuard keys
	privateKey, publicKey, err := wg.GenerateKeyPair()
	if err != nil {
		return "", fmt.Errorf("generating WireGuard keys: %w", err)
	}

	hostname, _ := os.Hostname()

	// Create WebRTC peer connection with STUN server
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{stunServer}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return "", fmt.Errorf("creating peer connection: %w", err)
	}
	defer func() { _ = pc.Close() }()

	// Data channel required to generate SDP offer (not used for data)
	if _, err := pc.CreateDataChannel("chat", nil); err != nil {
		return "", fmt.Errorf("creating data channel: %w", err)
	}

	// Create and set local offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("creating offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("setting local description: %w", err)
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	<-gatherComplete

	// Inject Amplifi tunnel attributes into SDP
	localSDP := InjectTunnelInfo(pc.LocalDescription().SDP, hostname, devicePlatform, publicKey)

	// Exchange SDP via Amplifi signaling
	iceServers, err := client.GetICEConfig(deviceToken)
	if err != nil {
		return "", fmt.Errorf("getting ICE config: %w", err)
	}

	answerSDP, err := client.SignalingConnect(localSDP, iceServers, deviceToken)
	if err != nil {
		return "", fmt.Errorf("signaling connect: %w", err)
	}

	// Parse Amplifi attributes from SDP answer
	attrs, err := ParseAmplifiAttributes(answerSDP)
	if err != nil {
		return "", fmt.Errorf("parsing SDP answer: %w", err)
	}

	// Register ICE callback before SetRemoteDescription to avoid race condition
	iceDone := make(chan error, 1)
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		switch state {
		case webrtc.ICEConnectionStateCompleted, webrtc.ICEConnectionStateConnected:
			select {
			case iceDone <- nil:
			default:
			}
		case webrtc.ICEConnectionStateFailed:
			select {
			case iceDone <- fmt.Errorf("ICE connection failed"):
			default:
			}
		}
	})

	// Set remote description to start ICE negotiation
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	}
	if err := pc.SetRemoteDescription(answer); err != nil {
		return "", fmt.Errorf("setting remote description: %w", err)
	}

	select {
	case err := <-iceDone:
		if err != nil {
			return "", err
		}
	case <-time.After(iceTimeout):
		return "", fmt.Errorf("ICE negotiation timed out")
	}

	// Extract candidate pair for WireGuard endpoint
	stats := pc.GetStats()
	var localPort uint16
	var remoteIP string
	var remotePort uint16

	for _, s := range stats {
		if cp, ok := s.(webrtc.ICECandidatePairStats); ok && cp.Nominated {
			// Find the local and remote candidate details
			for _, s2 := range stats {
				if lc, ok := s2.(webrtc.ICECandidateStats); ok && lc.ID == cp.LocalCandidateID {
					localPort = uint16(lc.Port)
				}
				if rc, ok := s2.(webrtc.ICECandidateStats); ok && rc.ID == cp.RemoteCandidateID {
					remoteIP = rc.IP
					remotePort = uint16(rc.Port)
				}
			}
		}
	}

	if remoteIP == "" || remotePort == 0 {
		return "", fmt.Errorf("could not extract ICE candidate pair")
	}

	// Generate WireGuard config
	configText := fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = %d
Address = %s/32
DNS = %s

[Peer]
PublicKey = %s
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = %s:%d`,
		privateKey,
		localPort,
		attrs.InterfaceAddr,
		attrs.DNSAddr,
		attrs.RemotePublicKey,
		remoteIP,
		remotePort,
	)

	return configText, nil
}
