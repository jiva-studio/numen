// Package modelcallcontract re-exports the port contract of the calls on models.
package modelcallcontract

import "github.com/jiva-studio/numen/libs/core/port/connectionport"

type (
	ModelCalls       = connectionport.ModelCalls
	ModelCallsHost   = connectionport.ModelCallsHost
	Notices          = connectionport.Notices
	Provider         = connectionport.Provider
	ProviderSource   = connectionport.ProviderSource
	ProviderCall     = connectionport.ProviderCall
	FailureReason    = connectionport.FailureReason
	TryNextError     = connectionport.TryNextError
	StopError        = connectionport.StopError
	CompleteRequest  = connectionport.CompleteRequest
	CompleteResult   = connectionport.CompleteResult
	Message          = connectionport.Message
	Model            = connectionport.Model
	ReasoningLevel   = connectionport.ReasoningLevel
	ModelProfile     = connectionport.ModelProfile
	EmbeddingProfile = connectionport.EmbeddingProfile
	AllFailedError   = connectionport.AllFailedError
	RejectedError    = connectionport.RejectedError
	Failure          = connectionport.Failure
)

const (
	FailureReasonUnknown         = connectionport.FailureReasonUnknown
	FailureReasonKeyRejected     = connectionport.FailureReasonKeyRejected
	FailureReasonPaymentRequired = connectionport.FailureReasonPaymentRequired
	FailureReasonRateLimited     = connectionport.FailureReasonRateLimited
	FailureReasonUnreachable     = connectionport.FailureReasonUnreachable
	FailureReasonTimeout         = connectionport.FailureReasonTimeout
	FailureReasonBadRequest      = connectionport.FailureReasonBadRequest
)

var ErrNoConnection = connectionport.ErrNoConnection
