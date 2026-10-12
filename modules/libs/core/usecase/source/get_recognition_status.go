package source

// RecognitionStatus is what stands in the way of reading a scan now.
type RecognitionStatus struct {
	IsRunning bool
	IsReady   bool
}

// GetRecognitionStatus says whether a document is being read and whether the
// models recognition needs are here.
type GetRecognitionStatus struct {
	worker *RecognitionWorker
}

func NewGetRecognitionStatus(worker *RecognitionWorker) *GetRecognitionStatus {
	return &GetRecognitionStatus{worker: worker}
}

func (u *GetRecognitionStatus) Execute() RecognitionStatus {
	return RecognitionStatus{IsRunning: u.worker.IsRunning(), IsReady: u.worker.Ready()}
}
