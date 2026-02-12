package wg

import qrcode "github.com/skip2/go-qrcode"

// GenerateQRString generates a terminal-printable QR code from an interface config.
func GenerateQRString(iface *Interface) (string, error) {
	conf := MarshalConfig(iface)
	qr, err := qrcode.New(conf, qrcode.Medium)
	if err != nil {
		return "", err
	}
	return qr.ToSmallString(false), nil
}
