package llm

// The names of the operations of the contract.
const (
	OperationComplete  = "llm.complete"
	OperationChat      = "llm.chat"
	OperationEmbed     = "llm.embed"
	OperationRerank    = "llm.rerank"
	OperationProfile   = "llm.profile"
	OperationModels    = "llm.models"
	OperationReasoning = "llm.reasoning"
)

// The keys of the row fields a connection type of the contract declares.
const (
	// OptionModel is the model the person chose for the row.
	OptionModel = "model"
	// OptionReasoning is the reasoning level of the row; empty asks for the default of the provider.
	OptionReasoning = "reasoning"
)
