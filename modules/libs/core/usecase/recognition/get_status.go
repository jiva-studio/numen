package recognition

type Status struct {
	IsRunning bool
	IsReady   bool
}

// GetStatus says whether a document is being read and whether the
// models recognition needs are here.
type GetStatus struct {
	worker *Worker
}

func NewGetStatus(worker *Worker) *GetStatus {
	return &GetStatus{worker: worker}
}

func (u *GetStatus) Execute() Status {
	return Status{IsRunning: u.worker.IsRunning(), IsReady: u.worker.Ready()}
}
