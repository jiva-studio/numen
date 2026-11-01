//go:build darwin

package trash

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// unsupported is the Cocoa error code for a volume that has no trash.
const unsupported = 3328

// selectors are the selectors this file sends. The runtime returns the same selector for a name
// registered again.
type selectors struct {
	alloc, begin, drain, fromUTF8, fileURL, manager, trashItem, code, described, utf8String objc.SEL
}

func registerSelectors() selectors {
	return selectors{
		alloc:      objc.RegisterName("alloc"),
		begin:      objc.RegisterName("init"),
		drain:      objc.RegisterName("drain"),
		fromUTF8:   objc.RegisterName("stringWithUTF8String:"),
		fileURL:    objc.RegisterName("fileURLWithPath:isDirectory:"),
		manager:    objc.RegisterName("defaultManager"),
		trashItem:  objc.RegisterName("trashItemAtURL:resultingItemURL:error:"),
		code:       objc.RegisterName("code"),
		described:  objc.RegisterName("localizedDescription"),
		utf8String: objc.RegisterName("UTF8String"),
	}
}

// loadFoundation loads the framework the classes named here come from. The loader counts each
// open of a library it holds, so a second call loads nothing again.
func loadFoundation() error {
	const frameworkPath = "/System/Library/Frameworks/Foundation.framework/Foundation"
	_, err := purego.Dlopen(frameworkPath, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	return err
}

// moveToTrash moves the folder to the Finder trash and records its origin for
// Put Back.
func moveToTrash(path string, _ time.Time) error {
	if err := loadFoundation(); err != nil {
		return err
	}
	sel := registerSelectors()

	// Every object made here is autoreleased, and this is the pool it drains
	// into. A goroutine carries none of its own.
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(sel.alloc).Send(sel.begin)
	defer pool.Send(sel.drain)

	name := objc.ID(objc.GetClass("NSString")).Send(sel.fromUTF8, path)
	folder := objc.ID(objc.GetClass("NSURL")).Send(sel.fileURL, name, true)

	var failure objc.ID
	moved := objc.Send[bool](
		objc.ID(objc.GetClass("NSFileManager")).Send(sel.manager),
		sel.trashItem, folder, objc.ID(0), unsafe.Pointer(&failure),
	)
	if moved {
		return nil
	}
	if failure != 0 && objc.Send[int](failure, sel.code) == unsupported {
		return fmt.Errorf("%s: %w", path, machineport.ErrNoTrash)
	}
	return fmt.Errorf("%s: %s", path, getFailureMessage(sel, failure))
}

// getFailureMessage returns the description Cocoa gives for the failure.
func getFailureMessage(sel selectors, failure objc.ID) string {
	if failure == 0 {
		return "the trash would not take it"
	}
	return objc.Send[string](failure.Send(sel.described), sel.utf8String)
}
