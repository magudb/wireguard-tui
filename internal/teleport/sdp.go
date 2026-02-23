package teleport

import (
	"fmt"
	"strings"
)

// AmplifiAttributes holds the Amplifi-specific values extracted from an SDP answer.
type AmplifiAttributes struct {
	InterfaceAddr   string
	DNSAddr         string
	RemotePublicKey string
}

// InjectTunnelInfo adds custom Amplifi attributes into an SDP string.
// Attributes are inserted after the "s=-" line, matching the Python implementation.
func InjectTunnelInfo(sdp, friendlyName, platform, publicKey string) string {
	attrs := strings.Join([]string{
		"a=tool:ubnt_webrtc version ",
		"a=uca_acf5_amplifi_friendly_name:" + friendlyName,
		"a=uca_acf5_amplifi_nomination_mode:slave",
		"a=uca_acf5_amplifi_platform:" + platform,
		"a=uca_acf5_amplifi_tunnel_pub_key:" + publicKey,
	}, "\r\n")

	parts := strings.SplitN(sdp, "s=-", 2)
	if len(parts) != 2 {
		return sdp
	}
	return parts[0] + "s=-" + "\r\n" + attrs + parts[1]
}

// ParseAmplifiAttributes extracts Amplifi-specific attributes from an SDP answer.
func ParseAmplifiAttributes(sdp string) (AmplifiAttributes, error) {
	var attrs AmplifiAttributes

	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "a=") {
			continue
		}

		// Parse "a=key:value"
		kv := strings.TrimPrefix(line, "a=")
		key, value, ok := strings.Cut(kv, ":")
		if !ok {
			continue
		}

		switch key {
		case "uca_acf5_amplifi_ipv4_addr":
			attrs.InterfaceAddr = value
		case "uca_acf5_amplifi_ipv4_dns_addr0":
			attrs.DNSAddr = value
		case "uca_acf5_amplifi_tunnel_pub_key":
			attrs.RemotePublicKey = value
		}
	}

	if attrs.InterfaceAddr == "" || attrs.DNSAddr == "" || attrs.RemotePublicKey == "" {
		return attrs, fmt.Errorf("missing required Amplifi attributes in SDP answer")
	}

	return attrs, nil
}
