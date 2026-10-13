package transcription

// GetReadiness says whether the models transcription needs are here.
type GetReadiness struct {
	worker *Worker
}

func NewGetReadiness(worker *Worker) *GetReadiness {
	return &GetReadiness{worker: worker}
}

func (u *GetReadiness) Execute() bool {
	return u.worker.Ready()
}
