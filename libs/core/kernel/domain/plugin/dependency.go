package plugin

// DependencyState is the install state of a dependency on this machine.
type DependencyState string

// The states a dependency is reported in.
const (
	DependencyStateMissing      DependencyState = "missing"
	DependencyStateProvisioning DependencyState = "provisioning"
	DependencyStateReady        DependencyState = "ready"
	DependencyStateFailed       DependencyState = "failed"
)

// ResolvedDependency is the install location of a ready dependency and how a process uses it.
type ResolvedDependency struct {
	ID          string
	RootPath    string
	PrimaryPath string
	EnvVars     map[string]string
	Metadata    map[string]string
}

// DependencyStatus is the state of one dependency at one moment.
type DependencyStatus struct {
	ID              string
	State           DependencyState
	BytesDownloaded int64
	BytesTotal      int64
	ProgressPercent float64
	Message         string
	Error           string
	Resolved        *ResolvedDependency
}

// RuntimeONNX is the id of the ONNX runtime a plugin requires as a runtime dependency.
const RuntimeONNX = "onnxruntime"
