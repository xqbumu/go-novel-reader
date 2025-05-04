package tts

import (
	"fmt"
	"os/exec"
	"runtime"
)

// SpeakAsync starts the macOS 'say' command asynchronously to read the given text aloud.
// It returns a channel that will receive an error if the command fails to start or finish,
// or nil if it completes successfully. The channel will be closed upon completion or error.
// Returns an immediate error if the OS is not macOS or text is empty.
func SpeakAsync(text string) (<-chan error, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("TTS functionality is only supported on macOS")
	}
	if text == "" {
		return nil, fmt.Errorf("cannot speak empty text")
	}

	cmd := exec.Command("say", text)
	err := cmd.Start() // Start the command asynchronously
	if err != nil {
		return nil, fmt.Errorf("failed to start 'say' command: %w", err)
	}

	doneChan := make(chan error, 1) // Buffered channel to avoid blocking sender

	// Goroutine to wait for the command to finish
	go func() {
		defer close(doneChan)
		waitErr := cmd.Wait() // Wait for the command to complete
		if waitErr != nil {
			doneChan <- fmt.Errorf("'say' command finished with error: %w", waitErr)
		} else {
			doneChan <- nil // Signal successful completion
		}
	}()

	return doneChan, nil // Return the channel for the caller to wait on
}

// Speak runs the 'say' command synchronously (waits for completion).
// This is kept for simplicity if async behavior is not needed.
func Speak(text string) error {
	doneChan, err := SpeakAsync(text)
	if err != nil {
		return err // Error starting the command
	}
	// Wait for the command to finish and get the result
	err = <-doneChan
	return err // Return the error from cmd.Wait() or nil
}

// TODO: Implement a way to stop ongoing speech.
// This would require storing the *exec.Cmd process associated with SpeakAsync
// and calling cmd.Process.Kill() or sending a signal. This adds complexity
// managing the process lifecycle.
