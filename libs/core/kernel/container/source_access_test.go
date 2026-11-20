package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
)

func TestSourceAccessFollowsTheActivePlugin(t *testing.T) {
	absent := getSourceAccess(nil)
	if absent.searchSource != nil || absent.watcher != nil || absent.queries != nil || absent.floorHint != nil {
		t.Fatal("the absent source supplied a plugin capability")
	}
	source := &sourcebridge.Source{}
	active := getSourceAccess(source)
	if active.searchSource != source || active.watcher != source || active.queries != source || active.floorHint == nil {
		t.Fatal("the opened source did not supply every plugin capability")
	}
}
