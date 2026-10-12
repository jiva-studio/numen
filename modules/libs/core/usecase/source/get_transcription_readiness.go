package source

// GetTranscriptionReadiness says whether the models transcription needs are here.
type GetTranscriptionReadiness struct {
	worker *TranscriptionWorker
}

func NewGetTranscriptionReadiness(worker *TranscriptionWorker) *GetTranscriptionReadiness {
	return &GetTranscriptionReadiness{worker: worker}
}

func (u *GetTranscriptionReadiness) Execute() bool {
	return u.worker.Ready()
}
