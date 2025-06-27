package runtime

import (
	"fmt"

	"github.com/opencode-ai/opencode/internal/orchestrator/runtime/kubernetes"
)

// DefaultFactory implements the Factory interface
type DefaultFactory struct{}

// NewDefaultFactory creates a new default runtime factory
func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{}
}

// CreateRuntime creates a new runtime instance based on the configuration
func (f *DefaultFactory) CreateRuntime(config Config) (Runtime, error) {
	switch config.GetType() {
	case "kubernetes":
		kubeConfig, ok := config.(*kubernetes.Config)
		if !ok {
			return nil, fmt.Errorf("invalid configuration type for kubernetes runtime")
		}
		if err := kubeConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid kubernetes configuration: %w", err)
		}
		return kubernetes.NewRuntime(kubeConfig)
	default:
		return nil, fmt.Errorf("unsupported runtime type: %s", config.GetType())
	}
}

// SupportedTypes returns the list of supported runtime types
func (f *DefaultFactory) SupportedTypes() []string {
	return []string{"kubernetes"}
}
