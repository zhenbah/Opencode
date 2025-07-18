package logging

import (
	"fmt"
	"log/slog"
	"os"

	// "path/filepath"
	"encoding/json"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// Global logging file writer
var globalLogFile *os.File
var globalLogMutex sync.Mutex

// InitGlobalLogging initializes global logging to the specified file
func InitGlobalLogging(logFilePath string) error {
	globalLogMutex.Lock()
	defer globalLogMutex.Unlock()

	// Close existing log file if open
	if globalLogFile != nil {
		globalLogFile.Close()
	}

	// Create/open the log file (truncate if exists)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open global log file: %w", err)
	}

	globalLogFile = file
	return nil
}

// logToGlobalFile writes a log message to the global log file
func logToGlobalFile(level, msg string, args ...any) {
	if globalLogFile == nil {
		return
	}

	globalLogMutex.Lock()
	defer globalLogMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	// Add args if present
	if len(args) > 0 {
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				logLine += fmt.Sprintf(" %v=%v", args[i], args[i+1])
			} else {
				logLine += fmt.Sprintf(" %v", args[i])
			}
		}
	}

	logLine += "\n"

	globalLogFile.WriteString(logLine)
	globalLogFile.Sync()
}

func getCaller() string {
	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		funcName := ""
		if fn != nil {
			funcName = fn.Name()
		}
		return fmt.Sprintf("%s:%d (%s)", file, line, funcName)
	}
	return "unknown"
}
func Info(msg string, args ...any) {
	source := getCaller()
	slog.Info(msg, append([]any{"source", source, "location", source}, args...)...)
	msg_with_source := fmt.Sprintf("%s [source: %s]", msg, source)
	logToGlobalFile("INFO", msg_with_source, args...)
}

func Debug(msg string, args ...any) {
	// slog.Debug(msg, args...)
	source := getCaller()
	slog.Debug(msg, append([]any{"source", source}, args...)...)
	logToGlobalFile("DEBUG", msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
	logToGlobalFile("WARN", msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
	logToGlobalFile("ERROR", msg, args...)
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
			defer file.Close()

			// Write panic information and stack trace
			fmt.Fprintf(file, "Panic in %s: %v\n\n", name, r)
			fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(file, "Stack Trace:\n%s\n", debug.Stack())

			InfoPersist(fmt.Sprintf("Panic details written to %s", filename))
		}

		// Execute cleanup function if provided
		if cleanup != nil {
			cleanup()
		}
	}
}

// Message Logging for Debug
var MessageDir string

func GetSessionPrefix(sessionId string) string {
	return sessionId[:8]
}

var sessionLogMutex sync.Mutex

func AppendToSessionLogFile(sessionId string, filename string, content string) string {
	if MessageDir == "" || sessionId == "" {
		return ""
	}
	sessionPrefix := GetSessionPrefix(sessionId)

	sessionLogMutex.Lock()
	defer sessionLogMutex.Unlock()

	sessionPath := fmt.Sprintf("%s/%s", MessageDir, sessionPrefix)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		if err := os.MkdirAll(sessionPath, 0o766); err != nil {
			Error("Failed to create session directory", "dirpath", sessionPath, "error", err)
			return ""
		}
	}

	filePath := fmt.Sprintf("%s/%s", sessionPath, filename)

	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Error("Failed to open session log file", "filepath", filePath, "error", err)
		return ""
	}
	defer f.Close()

	// Append chunk to file
	_, err = f.WriteString(content)
	if err != nil {
		Error("Failed to write chunk to session log file", "filepath", filePath, "error", err)
		return ""
	}
	return filePath
}

func WriteRequestMessageJson(sessionId string, requestSeqId int, message any) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	msgJson, err := json.Marshal(message)
	if err != nil {
		Error("Failed to marshal message", "session_id", sessionId, "request_seq_id", requestSeqId, "error", err)
		return ""
	}
	return WriteRequestMessage(sessionId, requestSeqId, string(msgJson))
}

func WriteRequestMessage(sessionId string, requestSeqId int, message string) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	filename := fmt.Sprintf("%d_request.json", requestSeqId)

	return AppendToSessionLogFile(sessionId, filename, message)
}

func AppendToStreamSessionLogJson(sessionId string, requestSeqId int, jsonableChunk any) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	chunkJson, err := json.Marshal(jsonableChunk)
	if err != nil {
		Error("Failed to marshal message", "session_id", sessionId, "request_seq_id", requestSeqId, "error", err)
		return ""
	}
	return AppendToStreamSessionLog(sessionId, requestSeqId, string(chunkJson))
}

func AppendToStreamSessionLog(sessionId string, requestSeqId int, chunk string) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	filename := fmt.Sprintf("%d_response_stream.log", requestSeqId)
	return AppendToSessionLogFile(sessionId, filename, chunk)
}

func WriteChatResponseJson(sessionId string, requestSeqId int, response any) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	responseJson, err := json.Marshal(response)
	if err != nil {
		Error("Failed to marshal response", "session_id", sessionId, "request_seq_id", requestSeqId, "error", err)
		return ""
	}
	filename := fmt.Sprintf("%d_response.json", requestSeqId)

	return AppendToSessionLogFile(sessionId, filename, string(responseJson))
}

func WriteToolResultsJson(sessionId string, requestSeqId int, toolResults any) string {
	if MessageDir == "" || sessionId == "" || requestSeqId <= 0 {
		return ""
	}
	toolResultsJson, err := json.Marshal(toolResults)
	if err != nil {
		Error("Failed to marshal tool results", "session_id", sessionId, "request_seq_id", requestSeqId, "error", err)
		return ""
	}
	filename := fmt.Sprintf("%d_tool_results.json", requestSeqId)
	return AppendToSessionLogFile(sessionId, filename, string(toolResultsJson))
}
