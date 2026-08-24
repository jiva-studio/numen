//go:build darwin

package trash

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// unsupported is the Cocoa error for a volume that keeps nothing a person
// deleted.
const unsupported = 3328

// The selectors this file sends, registered once. Registering one takes the
// runtime's global lock.
var (
	alloc      = objc.RegisterName("alloc")
	begin      = objc.RegisterName("init")
	drain      = objc.RegisterName("drain")
	fromUTF8   = objc.RegisterName("stringWithUTF8String:")
	fileURL    = objc.RegisterName("fileURLWithPath:isDirectory:")
	manager    = objc.RegisterName("defaultManager")
	trashItem  = objc.RegisterName("trashItemAtURL:resultingItemURL:error:")
	code       = objc.RegisterName("code")
	described  = objc.RegisterName("localizedDescription")
	utf8String = objc.RegisterName("UTF8String")
)

// foundation loads the framework the classes named here come from, once.
var foundation = sync.OnceValue(func() error {
	const at = "/System/Library/Frameworks/Foundation.framework/Foundation"
	_, err := purego.Dlopen(at, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	return err
})

// send puts the folder where the Finder shows what was deleted, recording where
// it came from so that Put Back works.
func send(path string) error {
	if err := foundation(); err != nil {
		return err
	}

	// Every object made here is autoreleased, and this is the pool it drains
	// into. A goroutine carries none of its own.
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(alloc).Send(begin)
	defer pool.Send(drain)

	name := objc.ID(objc.GetClass("NSString")).Send(fromUTF8, path)
	folder := objc.ID(objc.GetClass("NSURL")).Send(fileURL, name, true)

	var failure objc.ID
	moved := objc.Send[bool](
		objc.ID(objc.GetClass("NSFileManager")).Send(manager),
		trashItem, folder, objc.ID(0), unsafe.Pointer(&failure),
	)
	if moved {
		return nil
	}
	if failure != 0 && objc.Send[int](failure, code) == unsupported {
		return fmt.Errorf("%s: %w", path, port.ErrNoTrash)
	}
	return fmt.Errorf("%s: %s", path, reason(failure))
}

// reason is what Cocoa would tell a person about the failure.
func reason(failure objc.ID) string {
	if failure == 0 {
		return "the trash would not take it"
	}
	return objc.Send[string](failure.Send(described), utf8String)
}
