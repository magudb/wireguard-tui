package teleport

import (
	"strings"
	"testing"
)

func TestInjectTunnelInfo(t *testing.T) {
	sdp := "v=0\r\ns=-\r\nm=application 9 DTLS/SCTP 5000\r\n"
	result := InjectTunnelInfo(sdp, "myhost", "iOS", "pubkey123")

	if !strings.Contains(result, "a=uca_acf5_amplifi_friendly_name:myhost") {
		t.Error("missing friendly_name attribute")
	}
	if !strings.Contains(result, "a=uca_acf5_amplifi_tunnel_pub_key:pubkey123") {
		t.Error("missing tunnel_pub_key attribute")
	}
	if !strings.Contains(result, "a=uca_acf5_amplifi_platform:iOS") {
		t.Error("missing platform attribute")
	}
	if !strings.Contains(result, "a=uca_acf5_amplifi_nomination_mode:slave") {
		t.Error("missing nomination_mode attribute")
	}

	// Attributes should be inserted after "s=-"
	idx := strings.Index(result, "s=-")
	attrIdx := strings.Index(result, "a=uca_acf5_amplifi_friendly_name")
	if attrIdx < idx {
		t.Error("attributes should appear after s=- line")
	}
}

func TestParseAmplifiAttributes(t *testing.T) {
	sdp := "v=0\r\ns=-\r\n" +
		"a=uca_acf5_amplifi_ipv4_addr:10.64.0.5\r\n" +
		"a=uca_acf5_amplifi_ipv4_dns_addr0:192.168.1.1\r\n" +
		"a=uca_acf5_amplifi_tunnel_pub_key:routerPubKey123\r\n" +
		"m=application 9 DTLS/SCTP 5000\r\n"

	attrs, err := ParseAmplifiAttributes(sdp)
	if err != nil {
		t.Fatalf("ParseAmplifiAttributes() error: %v", err)
	}
	if attrs.InterfaceAddr != "10.64.0.5" {
		t.Errorf("InterfaceAddr = %q, want %q", attrs.InterfaceAddr, "10.64.0.5")
	}
	if attrs.DNSAddr != "192.168.1.1" {
		t.Errorf("DNSAddr = %q, want %q", attrs.DNSAddr, "192.168.1.1")
	}
	if attrs.RemotePublicKey != "routerPubKey123" {
		t.Errorf("RemotePublicKey = %q, want %q", attrs.RemotePublicKey, "routerPubKey123")
	}
}

func TestParseAmplifiAttributesMissing(t *testing.T) {
	sdp := "v=0\r\ns=-\r\n"
	_, err := ParseAmplifiAttributes(sdp)
	if err == nil {
		t.Error("expected error for missing attributes")
	}
}
