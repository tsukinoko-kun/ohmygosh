package history

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/config"
	"github.com/tsukinoko-kun/ohmygosh/internal/data"
)

var (
	filter    string
	peekIndex int = -1
)

type history []string

var historyFile = filepath.Join(data.Path, "history.txt")

func open() history {
	f, err := os.Open(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(data.Path, 0755)
			f, err = os.Create(historyFile)
			if err != nil {
				return nil
			}
		} else {
			return nil
		}
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func (h history) Close() {
	f, err := os.Create(historyFile)
	if err != nil {
		return
	}
	defer f.Close()

	for _, line := range h {
		if _, err := f.WriteString(line); err != nil {
			return
		}
		if _, err := f.WriteString("\n"); err != nil {
			return
		}
	}
	_ = f.Sync()
}

func Push(line string) {
	if config.Get.Shell.MaxHistoryLength == 0 {
		return
	}

	filter = ""
	peekIndex = -1

	h := open()
	if len(h) >= int(config.Get.Shell.MaxHistoryLength) {
		h = h[1:]
	}

	// delete equal lines
	for i := len(h) - 1; i >= 0; i-- {
		storedLine := h[i]
		if storedLine == line {
			h = append(h[:i], h[i+1:]...)
			break
		}
	}

	h = append(h, line)

	h.Close()
}

func SetFilter(s string) {
	filter = s
	peekIndex = -1
}

func Peek() string {
	if config.Get.Shell.MaxHistoryLength == 0 {
		return ""
	}

	h := open()
	if peekIndex < 0 {
		peekIndex = len(h)
	}
	for i := peekIndex - 1; i >= 0; i-- {
		if strings.Contains(h[i], filter) {
			peekIndex = i
			return h[i]
		}
	}

	if peekIndex > 0 && peekIndex < len(h) {
		return h[peekIndex]
	} else {
		return ""
	}
}

func PeekReverse() string {
	if config.Get.Shell.MaxHistoryLength == 0 {
		return ""
	}

	h := open()
	if peekIndex < 0 || peekIndex >= len(h) {
		peekIndex = -1
		return ""
	}
	for i := peekIndex + 1; i < len(h); i++ {
		if strings.Contains(h[i], filter) {
			peekIndex = i
			return h[i]
		}
	}

	peekIndex = -1
	return ""
}
