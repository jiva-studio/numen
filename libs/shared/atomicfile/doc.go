// Package atomicfile replaces a file atomically: a reader sees either the old
// content or the new content, never a partial write.
//
// The content goes to a temporary file in the folder of the target, is flushed,
// and is renamed over the target; the folder is flushed after the rename. On
// Windows the rename waits out a handle that another program holds on the target.
package atomicfile
