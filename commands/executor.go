package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Executor(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	} else if s == "quit" || s == "exit" {
		os.Exit(0)
		return
	}

	cmd := exec.Command("/bin/sh", "-c", "rdns-server "+s)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("got error: %s\n", err.Error())
	}
	return
}
