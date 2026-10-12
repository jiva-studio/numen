/**
 * Write queuing and dirty-field tracking for preset tabs.
 */
import { ref, type Ref } from 'vue'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Presets, Settings } from '../types'
import { WORDS as words } from '../words'

/** The pending write and the unwritten fields of one preset file. */
export interface WriteQueue {
  revision: string
  isWriting: boolean
  isWriteQueued: boolean
  activeWrite: Promise<void> | null
  hasWarned: boolean
  readonly dirtyFields: Set<keyof Settings>
  readonly hasExternalChange: Ref<boolean>
  readonly errorMessage: Ref<string>
}

export function createWriteQueue(): WriteQueue {
  return {
    revision: '',
    isWriting: false,
    isWriteQueued: false,
    activeWrite: null,
    hasWarned: false,
    dirtyFields: new Set<keyof Settings>(),
    hasExternalChange: ref(false),
    errorMessage: ref(''),
  }
}

/** Sends the current settings to the file the read came out of. */
export const sendWrite = async (
  queue: WriteQueue,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  writeMessage('')
  let answer
  try {
    answer = await core.write(path, settings, queue.revision)
  } catch {
    // The window says what it could not do; what the call carried back adds nothing a person can act on.
    queue.errorMessage.value = words.unwritten
    return
  }
  if (!answer.ok) {
    if (answer.error === 'changed') {
      queue.hasExternalChange.value = true
      return
    }
    queue.errorMessage.value = words.notSaved(answer.error)
    writeMessage(queue.errorMessage.value, 'error')
    return
  }
  queue.errorMessage.value = ''
  queue.revision = answer.value.at
  queue.dirtyFields.clear()
  queue.hasWarned = false
}

const runWrite = async (
  queue: WriteQueue,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  await sendWrite(queue, path, settings, core, writeMessage)
  queue.isWriting = false
  const again = queue.isWriteQueued && !queue.hasExternalChange.value
  queue.isWriteQueued = false
  if (again) {
    await requestWrite(queue, path, settings, core, writeMessage)
    return
  }
  queue.activeWrite = null
}

/** Requests a write, queuing behind the active write. */
export const requestWrite = (
  queue: WriteQueue,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  if (queue.isWriting) {
    queue.isWriteQueued = true
    return queue.activeWrite ?? Promise.resolve()
  }
  queue.isWriting = true
  const started = runWrite(queue, path, settings, core, writeMessage)
  queue.activeWrite = started
  return started
}

/** Flushes any unwritten changes or waits for the active write to finish. */
export const flushWrites = async (
  queue: WriteQueue,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  if (queue.dirtyFields.size > 0 && !queue.hasExternalChange.value) {
    await requestWrite(queue, path, settings, core, writeMessage)
  } else if (queue.activeWrite) {
    await queue.activeWrite
  }
}

/** Checks whether a tab can close, warning once if changes cannot be written. */
export const canCloseTab = async (
  queue: WriteQueue,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<boolean> => {
  await flushWrites(queue, path, settings, core, writeMessage)
  if (queue.dirtyFields.size === 0 && !queue.hasExternalChange.value) return true
  if (queue.hasWarned) return true
  queue.hasWarned = true
  return false
}
