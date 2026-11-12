/**
 * Write queuing and dirty-field tracking for preset tabs.
 */
import { ref, type Ref } from 'vue'
import { createWriteOrder, type WriteOrder } from '@numen/ui'
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
    readonly order: WriteOrder
  }
  readonly unsaved: {
    revision: string
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
      order: createWriteOrder(),
    },
    unsaved: { revision: '', hasWarned: false, dirtyFields: new Set<keyof Settings>() },
    status: { hasExternalChange: ref(false), errorMessage: ref('') },
  }
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
  queue.unsaved.revision = answer.value.at
  queue.unsaved.dirtyFields.clear()
  queue.unsaved.hasWarned = false
}

const runWrite = async (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  const settle = queue.inflight.order.beginWrite()
  try {
    await sendWrite(queue, options)
  } finally {
    settle()
  }
  queue.inflight.isWriting = false
  const again = queue.inflight.isWriteQueued && !queue.status.hasExternalChange.value
  const next = queue.inflight.queuedOptions ?? options
  queue.inflight.isWriteQueued = false
  queue.inflight.queuedOptions = null
  if (again) {
    await requestWrite(queue, next)
    return
  }
  queue.inflight.activeWrite = null
}

/** Requests a write, queuing behind the active write. */
export const requestWrite = (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  if (queue.inflight.isWriting) {
    queue.inflight.isWriteQueued = true
    queue.inflight.queuedOptions = options
    return queue.inflight.activeWrite ?? Promise.resolve()
  }
  queue.inflight.isWriting = true
  const started = runWrite(queue, options)
  queue.inflight.activeWrite = started
  return started
}

/** Flushes any unwritten changes or waits for the active write to finish. */
export const flushWrites = async (queue: WriteQueue, options: WriteOptions): Promise<void> => {
  if (queue.unsaved.dirtyFields.size > 0 && !queue.status.hasExternalChange.value) {
    await requestWrite(queue, options)
    // Stryker disable next-line ConditionalExpression: Settled write is null and awaiting null resolves immediately.
  } else if (queue.inflight.activeWrite) {
    await queue.inflight.activeWrite
  }
}

/** Checks whether a tab can close, warning once if changes cannot be written. */
export const canCloseTab = async (queue: WriteQueue, options: WriteOptions): Promise<boolean> => {
  await flushWrites(queue, options)
  if (queue.unsaved.dirtyFields.size === 0 && !queue.status.hasExternalChange.value) return true
  if (queue.unsaved.hasWarned) return true
  queue.unsaved.hasWarned = true
  return false
}
