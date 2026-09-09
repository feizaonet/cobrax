//go:build unix

package cobrax

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

// redirectStdio redirects os.Stdout/os.Stderr to the given pipe write ends at the
// fd level, so code holding init-time-captured *os.File references (e.g. Options.Out
// set to os.Stdout) also writes into the pipes. The returned restore function
// reinstates the original descriptors and is safe to call multiple times.
func redirectStdio(wOut, wErr *os.File) (restore func(), err error) {
	origStdoutFD, err := syscall.Dup(syscall.Stdout)
	if err != nil {
		return nil, fmt.Errorf("dup stdout: %w", err)
	}
	origStderrFD, err := syscall.Dup(syscall.Stderr)
	if err != nil {
		syscall.Close(origStdoutFD)
		return nil, fmt.Errorf("dup stderr: %w", err)
	}
	if err := syscall.Dup2(int(wOut.Fd()), syscall.Stdout); err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		return nil, fmt.Errorf("dup2 stdout: %w", err)
	}
	if err := syscall.Dup2(int(wErr.Fd()), syscall.Stderr); err != nil {
		syscall.Close(origStdoutFD)
		syscall.Close(origStderrFD)
		return nil, fmt.Errorf("dup2 stderr: %w", err)
	}

	var once sync.Once
	restore = func() {
		once.Do(func() {
			syscall.Dup2(origStderrFD, syscall.Stderr)
			syscall.Close(origStderrFD)
			syscall.Dup2(origStdoutFD, syscall.Stdout)
			syscall.Close(origStdoutFD)
		})
	}
	return restore, nil
}
