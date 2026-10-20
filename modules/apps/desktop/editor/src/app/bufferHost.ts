/**
 * The files open in a plugin's tabs, as a plugin is given the means to see them written out.
 */
import type { BufferHost } from '@numen/plugin-sdk'
import { raiseConflicts, type useFileFlush } from '@/features/file-conflict'

/** What the open files are written out and asked about through. */
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
      return {
        dispose: () => {
          remove()
        },
      }
    },
  }
}
