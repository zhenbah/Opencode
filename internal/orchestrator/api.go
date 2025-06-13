package orchestrator

import (
	"encoding/json"
	"net/http"

	"github.com/opencode-ai/opencode/internal/logging"
	"github.com/opencode-ai/opencode/internal/app" // For app.AppEvent and related types
)

// reportResultHandler handles incoming results from worker agents.
func reportResultHandler(service *orchestratorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var result WorkerResult
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			logging.Error("Failed to decode worker result from API", "error", err)
			http.Error(w, "Bad request: could not decode JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Log the received result
		logging.Info("Received worker result via API",
			"workerID", result.AgentID,
			"taskID", result.TaskID,
			"status", result.Status,
		)

		// Store the result
		service.mu.Lock()
		// Potentially check if a result for this workerID/taskID already exists
		// and decide on merging strategy or overwrite. For now, overwrite.
		service.workerResults[result.AgentID] = &result
		service.mu.Unlock()

		// Send WorkerCompletion event
		if service.eventQueue != nil {
			event := app.AppEvent{
				Type: app.EventTypeWorkerCompletion,
				Data: app.WorkerCompletionData{
					WorkerID: result.AgentID,
					Result:   result,
				},
			}
			// Non-blocking send to event queue; if full, log and drop event for now.
			// A more robust system might use a larger buffer or a different strategy.
			select {
			case service.eventQueue <- event:
				logging.Debug("WorkerCompletion event sent to queue", "workerID", result.AgentID, "taskID", result.TaskID)
			default:
				logging.Warn("Event queue full, WorkerCompletion event dropped", "workerID", result.AgentID, "taskID", result.TaskID)
			}
		} else {
			logging.Warn("Orchestrator eventQueue is nil, cannot send WorkerCompletion event", "workerID", result.AgentID)
		}


		// Additionally, if the worker command is still tracked, we might want to remove or update it.
		// This depends on whether cmd.Wait() in the stdout goroutine is still the primary mechanism for cleanup.
		// For now, the stdout goroutine will also try to store a result. The API one might be more up-to-date.
		// Consider adding a timestamp to WorkerResult and keeping the latest.

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Result received"})
	}
}
