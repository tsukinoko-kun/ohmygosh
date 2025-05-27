package shell

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tsukinoko-kun/ohmygosh/internal/config"
	ui "github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

var (
	Wd, _     = os.Getwd()
	ipcMut    = sync.Mutex{}
	ipcKey    = rand.Text()
	ipcLn     net.Listener
	ipcAddr   string
	ipcServer *http.Server
)

func ipcHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// check for valid key
	if r.Header.Get("X-Key") != ipcKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	ipcMut.Lock()
	defer ipcMut.Unlock()

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	line := strings.TrimSpace(string(buf))
	args := strings.SplitN(line, " ", 2)
	switch len(args) {
	case 2:
		// all good
	case 1:
		args = append(args, "")
	case 0:
		http.Error(w, "No command", http.StatusBadRequest)
		return
	}
	cmd := strings.TrimSpace(args[0])
	arg := strings.TrimSpace(args[1])
	switch cmd {
	case "exit":
		if arg == "" {
			ui.Exit(0)
		} else {
			if i, err := strconv.Atoi(arg); err == nil {
				ui.Exit(i)
			} else {
				ui.Exit(0)
			}
		}
	case "cd":
		if err := os.Chdir(arg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Wd = arg
	default:
		http.Error(w, "Unknown command", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func Init() {
	http.HandleFunc("/ipc", ipcHandler)
	ipcServer = &http.Server{Addr: ":0", Handler: http.DefaultServeMux}
	var err error
	ipcLn, err = net.Listen("tcp", ipcServer.Addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting IPC server: %v\n", err)
		os.Exit(1)
	}
	ipcAddr = fmt.Sprintf("http://localhost:%d", (ipcLn.Addr().(*net.TCPAddr)).Port)
	if err := ipcServer.Serve(ipcLn); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error serving IPC server: %v\n", err)
		os.Exit(1)
	}
}

func ClearIPC() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = ipcServer.Shutdown(ctx)
	_ = ipcLn.Close()
}

func GetShellName() string {
	return filepath.Base(config.Get.Shell.Exe)
}
