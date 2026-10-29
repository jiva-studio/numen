package engine

// ModelProfile holds the properties of an embedding model that the selected
// model determines.
type ModelProfile struct {
	Dimensions int
	MaxTokens  int
	Pooling    string
	// FloorHint is the starting cosine-similarity floor for the model. Zero means unknown.
	FloorHint float64
}

// Profile values shared by several model ids.
var (
	profileBGEM3 = ModelProfile{Dimensions: 1024, MaxTokens: 8192, Pooling: PoolHead, FloorHint: 0.30}

	profileOpenAISmall = ModelProfile{Dimensions: 1536, MaxTokens: 8191, Pooling: PoolHead, FloorHint: 0.30}
	profileOpenAILarge = ModelProfile{Dimensions: 3072, MaxTokens: 8191, Pooling: PoolHead, FloorHint: 0.30}
	profileOpenAIAda   = ModelProfile{Dimensions: 1536, MaxTokens: 8191, Pooling: PoolHead, FloorHint: 0.75}
)

// modelProfiles maps a model id to its profile. A model is listed under the id
// the settings use and under its canonical name.
var modelProfiles = map[string]ModelProfile{
	"Xenova/bge-m3":                 profileBGEM3,
	"bge-m3":                        profileBGEM3,
	"baai/bge-m3":                   profileBGEM3,
	"Xenova/multilingual-e5-small":  {Dimensions: 384, MaxTokens: 512, Pooling: PoolMean, FloorHint: 0.81},
	"Xenova/bge-small-en-v1.5":      {Dimensions: 384, MaxTokens: 512, Pooling: PoolHead, FloorHint: 0},
	"openai/text-embedding-3-small": profileOpenAISmall,
	"text-embedding-3-small":        profileOpenAISmall,
	"openai/text-embedding-3-large": profileOpenAILarge,
	"text-embedding-3-large":        profileOpenAILarge,
	"openai/text-embedding-ada-002": profileOpenAIAda,
	"text-embedding-ada-002":        profileOpenAIAda,
	"Xenova/all-MiniLM-L6-v2":       {Dimensions: 384, MaxTokens: 512, Pooling: PoolMean, FloorHint: 0},
	"Xenova/all-MiniLM-L12-v2":      {Dimensions: 384, MaxTokens: 512, Pooling: PoolMean, FloorHint: 0},
	"qwen/qwen3-embedding-8b":       {Dimensions: 4096, MaxTokens: 32768, Pooling: PoolHead, FloorHint: 0},
}

// GetModelProfile returns the profile of a model id. A model without an entry
// gets the BGE-M3 dimensions, token limit and pooling, and no floor hint.
func GetModelProfile(model string) ModelProfile {
	if profile, ok := modelProfiles[model]; ok {
		return profile
	}
	unknown := profileBGEM3
	unknown.FloorHint = 0
	return unknown
}

// HasModelProfile reports whether the table lists the model id.
func HasModelProfile(model string) bool {
	_, ok := modelProfiles[model]
	return ok
}
