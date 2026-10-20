import type { BufferHost, FileBuffers } from '../buffers'
import type { Disposable } from '../types'

// MockBufferHost is a BufferHost that holds what it is given, so a test can write the window out as a quit does.
export interface MockBufferHost extends BufferHost {
  // The buffers a plugin handed over, in the order it did.
  getTracked(): readonly FileBuffers[]
  // Writes every tracked buffer and every flush handler out.
  flush(): Promise<void>
}

// createMockBufferHost builds a buffer host that tracks buffers and flush handlers in memory.
export function createMockBufferHost(): MockBufferHost {
  const tracked = new Set<FileBuffers>()
  const handlers = new Set<() => Promise<unknown>>()
  const release = (release: () => void): Disposable => ({ dispose: release })
  return {
    track: (buffers) => {
      tracked.add(buffers)
      return release(() => tracked.delete(buffers))
    },
    addFlushHandler: (handler) => {
      handlers.add(handler)
      return release(() => handlers.delete(handler))
    },
    getTracked: () => [...tracked],
    flush: async () => {
      await Promise.all([...tracked].map((buffers) => buffers.flush()))
      await Promise.all([...handlers].map((handler) => handler()))
    },
  }
}
