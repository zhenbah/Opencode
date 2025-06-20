package logging

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"
)

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func InfoPersist(msg string, args ...any) {
	args = append(args, persistKeyArg, true)
	slog.Info(msg, args...)
}

func DebugPersist(msg string, args ...any) {
	args = append(args, persistKeyArg, true)
	slog.Debug(msg, args...)
}

func WarnPersist(msg string, args ...any) {
	args = append(args, persistKeyArg, true)
	slog.Warn(msg, args...)
}

func ErrorPersist(msg string, args ...any) {
	args = append(args, persistKeyArg, true)
	slog.Error(msg, args...)
}

// RecoverPanic is a common function to handle panics gracefully.
// It logs the error, creates a panic log file with stack trace,
// and executes an optional cleanup function before returning.
func RecoverPanic(name string, cleanup func()) {
	if r := recover(); r != nil {
		// Log the panic
		ErrorPersist(fmt.Sprintf("Panic in %s: %v", name, r))

		// Create a timestamped panic log file
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("opencode-panic-%s-%s.log", name, timestamp)

		file, err := os.Create(filename)
		if err != nil {
			ErrorPersist(fmt.Sprintf("Failed to create panic log: %v", err))
		} else {
			defer func() {
				if closeErr := file.Close(); closeErr != nil {
					ErrorPersist(fmt.Sprintf("Failed to close panic log file: %v", closeErr))
				}
			}()

			// Write panic information and stack trace
			if _, writeErr := fmt.Fprintf(file, "Panic in %s: %v\n\n", name, r); writeErr != nil {
				ErrorPersist(fmt.Sprintf("Failed to write panic info: %v", writeErr))
			}
			if _, writeErr := fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339)); writeErr != nil {
				ErrorPersist(fmt.Sprintf("Failed to write time info: %v", writeErr))
			}
			if _, writeErr := fmt.Fprintf(file, "Stack Trace:\n%s\n", debug.Stack()); writeErr != nil {
				ErrorPersist(fmt.Sprintf("Failed to write stack trace: %v", writeErr))
			}

			InfoPersist(fmt.Sprintf("Panic details written to %s", filename))
		}

		// Execute cleanup function if provided
		if cleanup != nil {
			cleanup()
		}
	}
}
