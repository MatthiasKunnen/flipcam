package flipcamlib

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func defaultString(value string, defaultVal string) string {
	if value == "" {
		return defaultVal
	}

	return value
}

func sudoCommand(ctx context.Context, cmd []string) *exec.Cmd {
	args := make([]string, len(cmd)+1)
	args[0] = "--non-interactive"
	copy(args[1:], cmd)
	fmt.Println(strings.Join(args, " "))
	return exec.CommandContext(ctx, "sudo", args...)
}
