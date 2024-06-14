package providers

import (
	"fmt"
	"github.com/gigurra/ai/common"
	"github.com/gigurra/ai/config"
	"github.com/gigurra/ai/domain"
	"github.com/gigurra/ai/providers/google_cloud_provider"
	"github.com/gigurra/ai/providers/openai_provider"
)

func CreateProvider(cfg config.Config) domain.Provider {
	switch cfg.Provider {
	case "openai":
		return openai_provider.NewOpenAIProvider(cfg.OpenAI, cfg.Verbose)
	case "google-cloud":
		return google_cloud_provider.NewGoogleCloudProvider(cfg.GoogleCloud, cfg.Verbose)
	default:
		common.FailAndExit(1, fmt.Sprintf("Unsupported provider: %s", cfg.Provider))
		return nil // needed to compile :S
	}
}
