//go:build linux
// +build linux

package neofetch

import (
	"os"
	"regexp"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

var Print = strings.Repeat("\n", 20)

func init() {
	go initLinux()
}

func readOneFileOf(names ...string) string {
	for _, name := range names {
		if content, err := os.ReadFile(name); err == nil {
			return string(content)
		}
	}
	return ""
}

func initLinux() {
	art := `            .-"""-.` + "\n" +
		"           '       \\" + "\n" +
		"          |,.  ,-.  |" + "\n" +
		"          |()L( ()| |" + "\n" +
		"          |,'  `\".| |" + "\n" +
		"          |.___.',| `" + "\n" +
		"         .j `--\"' `  `." + "\n" +
		"        / '        '   \\" + "\n" +
		"       / /          `   `." + "\n" +
		"      / /            `    ." + "\n" +
		"     / /              l   |" + "\n" +
		"    . ,               |   |" + "\n" +
		"    ,\"`.             .|   |" + "\n" +
		" _.'   ``.          | `..-'l" + "\n" +
		"|       `.`,        |      `." + "\n" +
		"|         `.    __.j         )" + "\n" +
		"|__        |--\"\"___|      ,-'" + "\n" +
		"   `\"--...,+\"\"\"\"   `._,.-' mh"

	Print = build(art, nil)

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}

	genericInit()

	Print = build(art, []data{
		{"OS", firstMatchOr(readOneFileOf("/etc/os-release", "/usr/lib/os-release"), "Unknown", regexp.MustCompile(`(?:PRETTY_)NAME="([^"]+)"\n`), regexp.MustCompile(`VERSION="([^"]+)"\n`))},
		{"Shell", shell},
		{"Terminal", terminal},
	})

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}
}
