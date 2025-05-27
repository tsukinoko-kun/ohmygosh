//go:build windows
// +build windows

package neofetch

import (
	"fmt"
	"golang.org/x/sys/windows"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

var Print = strings.Repeat("\n", 20)

func init() {
	go initWindows()
}

func initWindows() {
	const w = 10
	const h = 5
	art := strings.Repeat(red.Render(strings.Repeat("█", w))+"  "+green.Render(strings.Repeat("█", w))+"\n", h) +
		strings.Repeat(" ", w+w+2) + "\n" +
		strings.Repeat(blue.Render(strings.Repeat("█", w))+"  "+yellow.Render(strings.Repeat("█", w))+"\n", h)

	Print = build(art, nil)

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}

	genericInit()

	winVer, _ := windows.GetVersion()
	Print = build(art, []data{
		{"OS", fmt.Sprintf("Windows %d", winVer)},
		{"Shell", shell},
		{"DE", "Explorer"},
		{"Terminal", terminal},
	})

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}
}
