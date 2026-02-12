package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mlu/wireguard-tui/internal/tui"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "wireguard-tui must be run as root (use sudo)")
		os.Exit(1)
	}

	for _, bin := range []string{"wg", "wg-quick"} {
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Fprintf(os.Stderr, "Required binary not found: %s\n", bin)
			os.Exit(1)
		}
	}

	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
