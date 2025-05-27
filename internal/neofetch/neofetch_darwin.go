//go:build darwin
// +build darwin

package neofetch

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

var Print = strings.Repeat("\n", 20)

func init() {
	go initDarwin()
}

func initDarwin() {
	art := green.Render("\n                    'c.") + "\n" +
		green.Render("                 ,xNMM.") + "\n" +
		green.Render("               .OMMMMo") + "\n" +
		green.Render("               OMMM0,") + "\n" +
		green.Render("     .;loddo:' loolloddol;.") + "\n" +
		green.Render("   cKMMMMMMMMMMNWMMMMMMMMMM0:") + "\n" +
		yellow.Render(" .KMMMMMMMMMMMMMMMMMMMMMMMWd.") + "\n" +
		yellow.Render(" XMMMMMMMMMMMMMMMMMMMMMMMX.") + "\n" +
		red.Render(";MMMMMMMMMMMMMMMMMMMMMMMM:") + "\n" +
		red.Render(":MMMMMMMMMMMMMMMMMMMMMMMM:") + "\n" +
		red.Render(".MMMMMMMMMMMMMMMMMMMMMMMMX.") + "\n" +
		red.Render(" kMMMMMMMMMMMMMMMMMMMMMMMMWd.") + "\n" +
		magenta.Render(" .XMMMMMMMMMMMMMMMMMMMMMMMMMMk") + "\n" +
		magenta.Render("  .XMMMMMMMMMMMMMMMMMMMMMMMMK.") + "\n" +
		blue.Render("    kMMMMMMMMMMMMMMMMMMMMMMd") + "\n" +
		blue.Render("     ;KMMMMMMMWXXWMMMMMMMk.") + "\n" +
		blue.Render("       .cooc,.    .,coo:.")

	Print = build(art, nil)

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}

	genericInit()

	Print = build(art, []data{
		{"OS", fmt.Sprintf("macOS %s %s", runCommand("sw_vers", "-productVersion"), runCommand("sw_vers", "-buildVersion"))},
		{"Kernel", runCommand("uname", "-r")},
		{"Shell", shell},
		{"DE", "Aqua"},
		{"WM", "Quartz Compositor"},
		{"Terminal", terminal},
		{"CPU", fmt.Sprintf("%s (%d Cores)", runCommand("sysctl", "-n", "machdep.cpu.brand_string"), runtime.NumCPU())},
		{"GPU", firstMatchOr(runCommand("system_profiler", "SPDisplaysDataType"), "Unknown", regexp.MustCompile(` +Chipset Model: *([\w ]+)\n`))},
		{"Memory", fmt.Sprintf("%d GiB", atoiOr(runCommand("sysctl", "-n", "hw.memsize"), 0)/1024/1024/1024)},
	})

	if exit.P != nil {
		exit.P.Send(PrintUpdateMsg{})
	}
}
