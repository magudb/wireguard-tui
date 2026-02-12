package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "wireguard-tui must be run as root (use sudo)")
		os.Exit(1)
	}
	fmt.Println("wireguard-tui placeholder")
}
