import { ref, type Ref } from 'vue'
import type { MessageWriter } from '@numen/plugin-sdk'
import type { Presets, Settings } from '../../lib/presets'
import { WORDS as words } from '../lib/words'

/** The pending write and the unwritten fields of one preset file. */
export interface WriteQueue {
  readonly inflight: {
    isWriting: boolean
    isWriteQueued: boolean
    queuedOptions: WriteOptions | null
    activeWrite: Promise<void> | null
  }
  readonly unsaved: {
    revision: string
    version: bigint
    hasWarned: boolean
    readonly dirtyFields: Set<keyof Settings>
  }
  readonly status: {
    readonly hasExternalChange: Ref<boolean>
    readonly errorMessage: Ref<string>
  }
}

export function createWriteQueue(): WriteQueue {
  return {
    inflight: {
      isWriting: false,
      isWriteQueued: false,
      queuedOptions: null,
      activeWrite: null,
    },
    unsaved: {
      revision: '',
      version: 0n,
      hasWarned: false,
      dirtyFields: new Set<keyof Settings>(),
    },
    status: { hasExternalChange: ref(false), errorMessage: ref('') },
  }
}

/** Takes the answer of a read or write at a version unless an answer of a newer one was taken; returns whether it was taken. */
export const takeVersion = (queue: WriteQueue, version: bigint, revision: string): boolean => {
  if (version < queue.unsaved.version) return false
  queue.unsaved.version = version
  queue.unsaved.revision = revision
  return true
}

export interface WriteOptions {
  path: string
  settings: Settings
  core: Presets
  writeMessage: MessageWriter
}

/** Sends the current settings to the file the read came out of. */
export const sendWrite = async (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  const { path, settings, core, writeMessage } = options
  writeMessage('')
  let answer
  try {
    answer = await core.write(path, settings, queue.unsaved.revision)
  } catch {
    // The window states what failed; the error returned by the call adds nothing the user can act on.
    queue.status.errorMessage.value = words.unwritten
    return
  }
  if (!answer.ok) {
    if (answer.error === 'changed') {
      queue.status.hasExternalChange.value = true
      return
    }
    queue.status.errorMessage.value = words.notSaved(answer.error)
    writeMessage(queue.status.errorMessage.value, 'error')
    return
  }
  queue.status.errorMessage.value = ''
  takeVersion(queue, answer.value.version, answer.value.at)
  queue.unsaved.dirtyFields.clear()
  queue.unsaved.hasWarned = false
}
