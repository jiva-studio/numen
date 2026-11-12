/**
 * The files open in a plugin's tabs, as a plugin is given the means to see them written out.
 */
import { isUnsavedBuffersError, type BufferHost, type FileBuffers } from '@numen/plugin-loader'
import { raiseConflicts, type Notes } from './conflicts'
import type { Conflict, createFileFlush, FlushHandler } from './flush'

function hasNoteConflicts(buffers: FileBuffers): buffers is FileBuffers & Notes {
  return 'getOpenIds' in buffers
}

/** The file flush through which the open files are written and queried. */
export interface BufferHostDeps {
  readonly fileFlush: Pick<ReturnType<typeof createFileFlush>, 'addHandler' | 'raise'>
}

function settleUnsaved(flush: () => Promise<void>): FlushHandler {
  return async () => {
    try {
      await flush()
    } catch (error) {
      if (!isUnsavedBuffersError(error)) throw error
      return error.unsaved.map((one): Conflict => ({
        note: error.pluginId === '' ? one.name : `${error.pluginId}: ${one.name}`,
        keep: one.keep,
        take: one.take,
      }))
    }
  }
}

export function createBufferHost({ fileFlush }: BufferHostDeps): BufferHost {
  return {
    track: (buffers) => {
      const removeHandler = fileFlush.addHandler(settleUnsaved(buffers.flush))
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
      const remove = fileFlush.addHandler(settleUnsaved(handler))
      return {
        dispose: () => {
          remove()
        },
      }
    },
  }
}
