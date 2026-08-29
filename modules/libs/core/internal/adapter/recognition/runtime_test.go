package recognition

import (
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// A file that is there is not a library that loads, and every way of getting one
// ends in the same question: does it open.
func TestTheRuntimeThisMachineHoldsOpens(t *testing.T) {
	cfg := Defaults()
	cfg.Download = false
	engine, at, refused := opened(present(cfg))
	if engine == nil {
		t.Skip("this machine holds none:", refused)
	}
	defer engine.Destroy()
	t.Log("opened", at, engine.GetVersion())
}

func TestTheRuntimeFetchedForThisPlatformOpens(t *testing.T) {
	cfg := Defaults()
	address, err := runtimeAddress()
	if err != nil {
		t.Skip(err)
	}
	archive, err := fetched(t.Context(), cfg, address, false)
	if err != nil {
		t.Skip("nothing fetched on this machine:", err)
	}
	at, err := unpacked(archive, runtimeName())
	if err != nil {
		t.Fatal(err)
	}
	support()
	engine, err := ort.NewEngine(at)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Destroy()
	t.Log("opened", at, engine.GetVersion())
}
