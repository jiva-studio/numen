/**
 * The files open in a plugin's tabs, as a plugin is given the means to see them written out.
 */
import type { BufferHost, FileBuffers } from '@numen/plugin-sdk'
import { raiseConflicts, type createFileFlush, type Notes } from '@/features/file-conflict'

function hasNoteConflicts(buffers: FileBuffers): buffers is FileBuffers & Notes {
  return 'getOpenIds' in buffers
}

/** What the open files are written out and asked about through. */
export interface BufferHostDeps {
  readonly fileFlush: Pick<ReturnType<typeof createFileFlush>, 'addHandler' | 'raise'>
}

export function createBufferHost({ fileFlush }: BufferHostDeps): BufferHost {
  return {
    track: (buffers) => {
      const removeHandler = fileFlush.addHandler(buffers.flush)
      const stopRaising = hasNoteConflicts(buffers)
        ? raiseConflicts(buffers, fileFlush)
        : () => undefined // Buffers without note conflicts have nothing to stop raising.
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
