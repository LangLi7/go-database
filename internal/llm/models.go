package llm

// API endpoints
const (
	OpenRouterEndpoint       = "https://openrouter.ai/api/v1/chat/completions"
	OpenRouterModelsEndpoint = "https://openrouter.ai/api/v1/models"
	LMEndpointChat           = "/v1/chat/completions"
	LMEndpointModels         = "/api/v1/models"
	OllamaEndpoint           = "/api/generate"
)

// HTTP headers
const (
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"
	HeaderReferer       = "HTTP-Referer"
	RoleUser            = "user"
	RoleSystem          = "system"
)

// RemoteModel is a model from the OpenRouter API.
type RemoteModel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Pricing     ModelPricing `json:"pricing,omitempty"`
	ContextLen  int          `json:"context_length,omitempty"`
}

// ModelPricing holds per-token costs.
type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// IsFree returns true if the model has zero cost.
func (m RemoteModel) IsFree() bool {
	return m.Pricing.Completion == "0" && m.Pricing.Prompt == "0"
}

// LocalModel is a model from the LM Studio API.
type LocalModel struct {
	Key          string           `json:"key"`
	DisplayName  string           `json:"display_name"`
	Publisher    string           `json:"publisher"`
	Architecture string           `json:"architecture"`
	Quantization QuantInfo        `json:"quantization"`
	SizeBytes    int64            `json:"size_bytes"`
	ParamsString string           `json:"params_string"`
	Format       string           `json:"format"`
	Capabilities ModelCaps        `json:"capabilities"`
	Loaded       []LoadedInstance `json:"loaded_instances,omitempty"`
}

// QuantInfo describes model quantization.
type QuantInfo struct {
	Name          string `json:"name"`
	BitsPerWeight int    `json:"bits_per_weight"`
}

// ModelCaps describes model capabilities.
type ModelCaps struct {
	Vision            bool `json:"vision"`
	TrainedForToolUse bool `json:"trained_for_tool_use"`
}

// LoadedInstance describes a loaded model instance.
type LoadedInstance struct {
	Status string `json:"status"`
	PID    int    `json:"pid,omitempty"`
}
