package tts

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Speak uses the macOS 'say' command to read the given text aloud.
// Returns an error if the OS is not macOS or if the command fails.
func Speak(text string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("TTS functionality is only supported on macOS")
	}

	// Basic check for empty text
	if text == "" {
		return fmt.Errorf("cannot speak empty text")
	}

	cmd := exec.Command("say", text)
	// We run this synchronously for now.
	// For stopping capability, we might need cmd.Start() and manage the process.
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute 'say' command: %w", err)
	}

	return nil
}

// TODO: Implement a way to stop ongoing speech if needed.
// This might involve starting the 'say' command with cmd.Start()
// and keeping track of the process to kill it later (e.g., killall say).
