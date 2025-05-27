package term

import (
	"fmt"
	"net/url"
	"os"

	"github.com/tsukinoko-kun/ohmygosh/internal/paths"
)

var (
	hostname string
)

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
}

// SetSessionTitle sets the terminal session title using an OSC escape sequence.
func SetSessionTitle(title string) string {
	// OSC 2 sets both the icon name and window title.
	// The sequence is: ESC ] 2 ; <string> BEL
	// ESC (Escape) is '\x1b'
	// BEL (Bell) is '\x07'
	// The string terminator can also be ST (String Terminator), which is ESC \
	// For simplicity, BEL is often used and widely supported.
	return fmt.Sprintf("\x1b]2;%s\x07", title)
}

// PromptEnd returns OSC 133;A (End of Prompt / Ready for input / Save to close)
const PromptEnd = "\x1b]133;A\x07" // Using BEL as terminator

// CWDReportString generates the OSC 7 escape sequence to report the
// current working directory to the terminal emulator.
// The path should be an absolute path.
func CWDReportString(absPath string) string {
	// OSC 7 (Operating System Command 7) is the standard for reporting the
	// current working directory.
	// The format is: ESC ] 7 ; file://HOSTNAME/PATH BEL
	// ESC is \x1b (0x1B)
	// BEL is \x07 (0x07)
	// PATH needs to be URL-encoded, especially spaces.
	// Hostname is often omitted, making the format: ESC ] 7 ; file:///PATH BEL

	// URL-encode the path to handle spaces and other special characters.
	encodedPath := url.PathEscape(paths.Unixify(absPath))

	// Construct the full URL for OSC 7.
	// Note the triple slash for local file paths: file:///path/to/file
	// Or file://hostname/path/to/file
	fileURL := fmt.Sprintf("file://%s/%s", hostname, encodedPath)

	// Combine into the full escape sequence.
	// \x1b is the Escape character.
	// \x07 is the Bell character, which typically terminates the OSC sequence.
	return fmt.Sprintf("\x1b]7;%s\x07", fileURL)
}
