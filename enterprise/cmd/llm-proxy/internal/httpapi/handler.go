package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/llm-proxy/internal/events"
)

type Config struct {
	AnthropicAccessToken string
	OpenAIAccessToken    string
	OpenAIOrgID          string
}

func NewHandler(logger log.Logger, eventLogger events.Logger, config *Config) http.Handler {
	r := mux.NewRouter()

	// V1 service routes
	v1router := r.PathPrefix("/v1").Subrouter()
	if config.AnthropicAccessToken != "" {
		v1router.Handle("/completions/anthropic", newAnthropicHandler(logger, eventLogger, config.AnthropicAccessToken)).Methods(http.MethodPost)
	}
	if config.OpenAIAccessToken != "" {
		v1router.Handle("/completions/openai", newOpenAIHandler(logger, eventLogger, config.OpenAIAccessToken, config.OpenAIOrgID)).Methods(http.MethodPost)
	}

	return r
}
