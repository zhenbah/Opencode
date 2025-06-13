package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http" // Added for API server
	"os"
	"os/exec"
	"sync"
	"time" // Added for server shutdown

	"github.com/google/uuid"
	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/app" // For app.AppEvent
)

type TaskDefinition struct {
	Prompt string
	TaskID string // New: To be passed to worker & used in output
	// Potentially other fields later, like files to make available
}

// WorkerResult matches the structure the worker prints to stdout
type WorkerResult struct {
	AgentID         string `json:"agent_id"`
	TaskID          string `json:"task_id"` // Added
	TaskFilePath    string `json:"task_file_path"`
	OrchestratorAPI string `json:"orchestrator_api"`
	Status          string `json:"status"`
	Result          string `json:"result,omitempty"`
	Error           string `json:"error,omitempty"`
}

type Service interface {
	SpawnWorker(ctx context.Context, taskDef TaskDefinition) (workerID string, err error)
	GetWorkerResult(workerID string) (*WorkerResult, bool)
	StartAPIServer() error
	StopAPIServer(ctx context.Context) error
	// Other methods for managing workers might be added later
}

type orchestratorService struct {
	activeWorkers        map[string]*exec.Cmd
	workerResults        map[string]*WorkerResult
	mu                   sync.Mutex
	opencodeExecutablePath string
	apiServer            *http.Server   // Added for API server
	apiPort              string         // Added for API server port
	eventQueue           chan app.AppEvent // Added for event system
}

func NewService(opencodeExecutablePath string, eventQueue chan app.AppEvent) Service {
	return &orchestratorService{
		activeWorkers:        make(map[string]*exec.Cmd),
		workerResults:        make(map[string]*WorkerResult),
		opencodeExecutablePath: opencodeExecutablePath,
		apiPort:              "12345", // Default API port
		eventQueue:           eventQueue, // Store the event queue
	}
}

func (o *orchestratorService) StartAPIServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/report_result", reportResultHandler(o)) // reportResultHandler is in api.go

	o.apiServer = &http.Server{
		Addr:    ":" + o.apiPort,
		Handler: mux,
	}

	logging.Info("Orchestrator API server starting", "port", o.apiPort)
	go func() {
		if err := o.apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Using log.Printf here as logging.Error might not be available if logger itself failed,
			// or to avoid circular dependencies if logger uses orchestrator.
			// For production, ensure robust logging.
			log.Printf("Orchestrator API server ListenAndServe error: %v", err)
			// Consider a mechanism to signal this critical failure back to the main app.
		}
	}()
	return nil // Indicates setup was fine, actual error in goroutine.
}

func (o *orchestratorService) StopAPIServer(ctx context.Context) error {
	if o.apiServer == nil {
		return nil // Server not started
	}
	logging.Info("Orchestrator API server shutting down")
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Give 5 seconds to shutdown
	defer cancel()
	return o.apiServer.Shutdown(shutdownCtx)
}

func (o *orchestratorService) SpawnWorker(ctx context.Context, taskDef TaskDefinition) (workerID string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	workerID = uuid.NewString()

	// Create a temporary task file
	taskPayload := map[string]string{
		"task_prompt": taskDef.Prompt,
		"task_id":     taskDef.TaskID,
	}
	taskData, err := json.Marshal(taskPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task definition: %w", err)
	}

	tempTaskFile, err := os.CreateTemp("", "task-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp task file: %w", err)
	}
	defer os.Remove(tempTaskFile.Name()) // Clean up the file afterwards

	if _, err := tempTaskFile.Write(taskData); err != nil {
		tempTaskFile.Close() // Close explicitly on error before removing
		return "", fmt.Errorf("failed to write to temp task file: %w", err)
	}
	if err := tempTaskFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp task file: %w", err)
	}

	args := []string{
		"--worker-mode",
		"--agent-id", workerID,
		"--task-id", taskDef.TaskID,
		"--task-file", tempTaskFile.Name(),
		"--orchestrator-api", fmt.Sprintf("http://localhost:%s/report_result", o.apiPort), // Use configured port
	}

	cmd := exec.CommandContext(ctx, o.opencodeExecutablePath, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start worker process: %w", err)
	}

	o.activeWorkers[workerID] = cmd
	logging.Info("Worker process started", "workerID", workerID, "pid", cmd.Process.Pid, "taskID", taskDef.TaskID)

	go func() {
		// This goroutine now primarily handles logging stdout/stderr and process completion.
		// Result reporting is primarily done via API by the worker.
		// stdout can still be a fallback or for auxiliary data.

		// This goroutine now primarily handles logging stdout/stderr and process completion.
		// Result reporting is primarily done via API by the worker.
		// stdout can still be a fallback or for auxiliary data.

		stdoutData, err := io.ReadAll(stdoutPipe)
		if err != nil {
			logging.Error("Error reading worker stdout pipe", "workerID", workerID, "taskID", taskDef.TaskID, "error", err)
		} else if len(stdoutData) > 0 {
			logging.Info("Worker stdout", "workerID", workerID, "taskID", taskDef.TaskID, "data", string(stdoutData))
		}

		// stderrPipe, err := cmd.StderrPipe() // Optionally capture stderr too
		// if err != nil { ... }
		// stderrData, err := io.ReadAll(stderrPipe)
		// if err != nil { ... } else if len(stderrData) > 0 { logging.Info(...) }

		waitErr := cmd.Wait() // Wait for the command to complete. This must be after all pipe reads.

		o.mu.Lock()
		delete(o.activeWorkers, workerID)
		// Check if a result was already reported via API.
		// If not, or if cmd.Wait() indicates an error, we might want to store/update a status.
		existingResult, found := o.workerResults[workerID]
		if !found { // No result reported via API
			status := "completed_without_api_report"
			errMsg := ""
			if waitErr != nil {
				status = "failed_execution"
				errMsg = waitErr.Error()
				logging.Error("Worker process execution error (no API report)", "workerID", workerID, "taskID", taskDef.TaskID, "error", waitErr)
			}
			o.workerResults[workerID] = &WorkerResult{
				AgentID: workerID,
				TaskID:  taskDef.TaskID,
				Status:  status,
				Error:   errMsg,
			}
		} else { // Result was reported via API
			if waitErr != nil {
				// Process exited with an error after reporting via API.
				// Update status if it was 'completed' or not yet set by an error.
				if existingResult.Status == "completed" || existingResult.Status == "" || (existingResult.Status != "failed" && existingResult.Status != "failed_parsing_output" && existingResult.Status != "failed_execution") {
					existingResult.Status = "completed_with_execution_error"
					if existingResult.Error == "" {
						existingResult.Error = waitErr.Error()
					} else {
						existingResult.Error = fmt.Sprintf("%s; execution_error: %v", existingResult.Error, waitErr)
					}
				}
				logging.Error("Worker process execution error (API report was received)", "workerID", workerID, "taskID", taskDef.TaskID, "error", waitErr, "api_status", existingResult.Status)
			}
			// If waitErr is nil, and result was found, it means API report was successful and process exited cleanly.
			// The result from API is considered authoritative.
		}
		o.mu.Unlock()

		logging.Info("Worker process finished processing", "workerID", workerID, "taskID", taskDef.TaskID, "wait_error", waitErr)
	}()

	return workerID, nil
}

func (o *orchestratorService) GetWorkerResult(workerID string) (*WorkerResult, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	result, found := o.workerResults[workerID]
	return result, found
}
