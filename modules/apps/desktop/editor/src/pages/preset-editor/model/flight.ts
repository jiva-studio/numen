/**
 * Write queuing, flight synchronization, and dirty state tracking for preset tabs.
 */
import { ref, type Ref } from 'vue'
import type { MessageWriter } from '@/shared/notices/messages'
import type { Presets, Settings } from '../types'
import { WORDS as words } from '../words'

/** State tracking write flight and dirty settings for one preset file. */
export interface WriteFlight {
  at: string
  writing: boolean
  isWriteQueued: boolean
  flight: Promise<void> | null
  isWarned: boolean
  readonly theirs: Set<keyof Settings>
  readonly hasChanged: Ref<boolean>
  readonly errorMessage: Ref<string>
}

export function createWriteFlight(): WriteFlight {
  return {
    at: '',
    writing: false,
    isWriteQueued: false,
    flight: null,
    isWarned: false,
    theirs: new Set<keyof Settings>(),
    hasChanged: ref(false),
    errorMessage: ref(''),
  }
}

/** Sends the current settings to the file the read came out of. */
export const sendWrite = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  writeMessage('')
  let answer
  try {
    answer = await core.write(path, settings, flight.at)
  } catch {
    flight.errorMessage.value = words.unwritten
    return
  }
  if (answer.changed) {
    flight.hasChanged.value = true
    return
  }
  const writeError = answer.error
  if (writeError) {
    flight.errorMessage.value = words.notSaved(writeError)
    writeMessage(flight.errorMessage.value, 'error')
    return
  }
  flight.errorMessage.value = ''
  flight.at = answer.at
  flight.theirs.clear()
  flight.isWarned = false
}

const executeFlight = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  await sendWrite(flight, path, settings, core, writeMessage)
  flight.writing = false
  const again = flight.isWriteQueued && !flight.hasChanged.value
  flight.isWriteQueued = false
  if (again) {
    await requestWrite(flight, path, settings, core, writeMessage)
    return
  }
  flight.flight = null
}

/** Requests a write, queuing behind any write currently in flight. */
export const requestWrite = (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  if (flight.writing) {
    flight.isWriteQueued = true
    return flight.flight ?? Promise.resolve()
  }
  flight.writing = true
  const inFlight = executeFlight(flight, path, settings, core, writeMessage)
  flight.flight = inFlight
  return inFlight
}

/** Flushes any unwritten changes or waits for writes in flight to settle. */
export const flushWrites = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<void> => {
  if (flight.theirs.size > 0 && !flight.hasChanged.value) {
    await requestWrite(flight, path, settings, core, writeMessage)
  } else if (flight.flight) {
    await flight.flight
  }
}

/** Checks whether a tab can close, warning once if changes cannot be written. */
export const canCloseTab = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  writeMessage: MessageWriter,
): Promise<boolean> => {
  await flushWrites(flight, path, settings, core, writeMessage)
  if (flight.theirs.size === 0 && !flight.hasChanged.value) return true
  if (flight.isWarned) return true
  flight.isWarned = true
  return false
}
