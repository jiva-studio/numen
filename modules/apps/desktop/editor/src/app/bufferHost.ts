/**
 * The files the window's tabs hold open, as a plugin is given the means to see them written out.
 */
import type { BufferHost } from '@numen/plugin-sdk'
import { raiseConflicts, type useFileFlush } from '@/features/file-conflict'

/** What the files held open are written out and asked about through. */
export interface BufferHostDeps {
  readonly fileFlush: Pick<ReturnType<typeof useFileFlush>, 'addHandler' | 'raise'>
}

export function createBufferHost({ fileFlush }: BufferHostDeps): BufferHost {
  return {
    track: (buffers) => {
      const removeHandler = fileFlush.addHandler(buffers.flush)
      const stopRaising = raiseConflicts(buffers, fileFlush)
      return {
        dispose: () => {
          removeHandler()
          stopRaising()
        },
      }
    },
    addFlushHandler: (handler) => {
      const remove = fileFlush.addHandler(handler)
      return { dispose: () => void remove() }
    },
  }
}
