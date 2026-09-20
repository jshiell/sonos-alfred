// Command sonos-alfred is the Sonos workflow for Alfred. Alfred runs it as `filter <query>`, `do <action>` and,
// through filter, `refresh`.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"sonos-alfred/alfredjson"
	"sonos-alfred/app"
	"sonos-alfred/hub"
	"sonos-alfred/sonos"
)

const (
	refreshTimeout = 20 * time.Second
	doTimeout      = 10 * time.Second
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

// run does what the command line asks. Alfred shows whatever is written to stdout, so every failure is written
// there: a Script Filter must always print JSON, and the notification after `do` shows its output.
func run(args []string, stdout io.Writer) (exitCode int) {
	command := ""
	if len(args) > 0 {
		command = args[0]
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			exitCode = fail(command, stdout, fmt.Errorf("unexpected error: %v", recovered))
		}
	}()

	env, err := newEnv()
	if err != nil {
		return fail(command, stdout, err)
	}
	argument := ""
	if len(args) > 1 {
		argument = args[1]
	}

	switch command {
	case "filter":
		out, err := app.Filter(env, argument)
		if err != nil {
			return fail(command, stdout, err)
		}
		stdout.Write(out)
	case "refresh":
		ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
		defer cancel()
		if err := app.Refresh(ctx, env); err != nil {
			return fail(command, stdout, err)
		}
	case "do":
		ctx, cancel := context.WithTimeout(context.Background(), doTimeout)
		defer cancel()
		if err := app.Do(ctx, env, argument); err != nil {
			return fail(command, stdout, err)
		}
	default:
		return fail(command, stdout, fmt.Errorf("unknown command %q, want filter, do or refresh", command))
	}
	return 0
}

// fail reports err the way Alfred can show it.
func fail(command string, stdout io.Writer, err error) int {
	if command == "filter" {
		rows := []hub.Item{{Title: "Sonos hit a problem", Subtitle: err.Error()}}
		if out, renderErr := alfredjson.Render(rows, false); renderErr == nil {
			stdout.Write(out)
		}
		return 0 // a Script Filter that exits non-zero shows nothing at all
	}
	fmt.Fprintln(stdout, err)
	return 1
}

// newEnv builds what the commands run with. It is a variable so that tests can supply their own.
var newEnv = workflowEnv

func workflowEnv() (app.Env, error) {
	dir, err := dataDir()
	if err != nil {
		return app.Env{}, err
	}
	return app.Env{
		Dir:      dir,
		Now:      time.Now,
		Spawn:    spawnRefresh,
		Discover: discover,
		Connect:  func(host string) sonos.Backend { return sonos.NewClient("http://" + host + ":1400") },
	}, nil
}

// dataDir is where Alfred keeps this workflow's data, falling back to the user cache directory outside Alfred.
func dataDir() (string, error) {
	dir := os.Getenv("alfred_workflow_data")
	if dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cache, "sonos-alfred")
	}
	return dir, os.MkdirAll(dir, 0o755)
}

// discover finds a speaker: the one named by the SONOS_HOST workflow variable if set, else by SSDP.
func discover(ctx context.Context) (string, error) {
	if host := os.Getenv("SONOS_HOST"); host != "" {
		return host, nil
	}
	return sonos.Discover(ctx)
}

// spawnRefresh starts `refresh` as a separate process that outlives this one. It gets its own process group and no
// stdio, so Alfred, which waits for our stdout to close, doesn't wait for it.
func spawnRefresh() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	refresh := exec.Command(self, "refresh")
	refresh.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := refresh.Start(); err != nil {
		return err
	}
	return refresh.Process.Release()
}
