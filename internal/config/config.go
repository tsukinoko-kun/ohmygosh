package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/goccy/go-yaml"
)

type (
	Config struct {
		Shell Shell `yaml:"shell"`
		Ui    Ui    `yaml:"ui"`
	}

	Shell struct {
		Exe              string            `yaml:"executable"`
		Args             []string          `yaml:"args"`
		Alias            map[string]string `yaml:"alias"`
		Env              map[string]string `yaml:"env"`
		Completion       string            `yaml:"completion"`
		MaxHistoryLength uint              `yaml:"max_history_length"`
	}

	Ui struct {
		NormalColorBg string `yaml:"normal_mode_color"`
		NormalColorFg string `yaml:"-"`
		InsertColorBg string `yaml:"insert_mode_color"`
		InsertColorFg string `yaml:"-"`
		VisualColorBg string `yaml:"visual_mode_color"`
		VisualColorFg string `yaml:"-"`

		TextColor         string `yaml:"text_color"`
		CursorColor       string `yaml:"cursor_color"`
		CursorColorText   string `yaml:"cursor_text_color"`
		VisualSelectionBg string `yaml:"visual_selection_color"`

		BorderColor               string `yaml:"border_color"`
		BorderColorFocus          string `yaml:"border_focus_color"`
		HeaderColor               string `yaml:"header_color"`
		HeaderCommandColorRunning string `yaml:"header_command_running_color"`
		HeaderCommandColorDone    string `yaml:"header_command_done_color"`
		HeaderCommandColorFailed  string `yaml:"header_command_failed_color"`
		HeaderCopyColor           string `yaml:"header_copy_color"`

		RunningColor   string `yaml:"running_color"`
		CompletedColor string `yaml:"completed_color"`
		FailedColor    string `yaml:"failed_color"`
	}
)

var (
	ConfigDir  string
	ConfigFile string
)

var Get Config

var Environ []string

func Default() Config {
	shell := GetSystemShell()
	return Config{
		Shell: Shell{
			Exe:              shell,
			Completion:       shell,
			Args:             DefaultArgs(filepath.Base(shell)),
			MaxHistoryLength: 100,
		},
		Ui: Ui{
			NormalColorBg:             "4",
			NormalColorFg:             "0",
			InsertColorBg:             "2",
			InsertColorFg:             "0",
			VisualColorBg:             "5",
			VisualColorFg:             "0",
			TextColor:                 "7",
			CursorColor:               "7",
			CursorColorText:           "0",
			VisualSelectionBg:         "8",
			BorderColor:               "8",
			BorderColorFocus:          "5",
			HeaderColor:               "8",
			HeaderCommandColorRunning: "7",
			HeaderCommandColorDone:    "8",
			HeaderCommandColorFailed:  "1",
			HeaderCopyColor:           "8",
			RunningColor:              "4",
			CompletedColor:            "2",
			FailedColor:               "1",
		},
	}
}

func (c *Config) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	err = yaml.NewDecoder(f).Decode(c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
	return enc.Encode(c)
}

func init() {
	if xdgConfigHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		ConfigDir = filepath.Join(xdgConfigHome, "ohmygosh")
	} else {
		ConfigDir = defaultConfigDir()
	}
	ConfigFile = filepath.Join(ConfigDir, "config.yaml")

	if _, err := os.Stat(ConfigFile); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(ConfigDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
				os.Exit(1)
			}
			Get = Default()
			if err := Get.Save(ConfigFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := Get.Load(ConfigFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}

		if Get.Shell.Completion == "" {
			Get.Shell.Completion = Get.Shell.Exe
		}

		switch Get.Ui.NormalColorBg {
		case "0":
			Get.Ui.NormalColorFg = "7"
		case "", "default":
			Get.Ui.NormalColorBg = "4"
			fallthrough
		default:
			Get.Ui.NormalColorFg = "0"
		}

		switch Get.Ui.InsertColorBg {
		case "0":
			Get.Ui.InsertColorFg = "7"
		case "", "default":
			Get.Ui.InsertColorBg = "2"
			fallthrough
		default:
			Get.Ui.InsertColorFg = "0"
		}

		switch Get.Ui.VisualColorBg {
		case "0":
			Get.Ui.VisualColorFg = "7"
		case "", "default":
			Get.Ui.VisualColorBg = "5"
			fallthrough
		default:
			Get.Ui.VisualColorFg = "0"
		}
	}

	Environ = os.Environ()
	for k, v := range Get.Shell.Env {
		Environ = append(Environ, fmt.Sprintf("%s=%s", k, os.ExpandEnv(v)))
	}
}

var knownShellsLogin = []string{
	"bash",
	"zsh",
	"dash",
	"fish",
}

func DefaultArgs(shell string) []string {
	if slices.Contains(knownShellsLogin, shell) {
		return []string{"-l"}
	}
	return nil
}
