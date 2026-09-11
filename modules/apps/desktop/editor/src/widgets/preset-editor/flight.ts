/**
 * Write queuing, flight synchronization, and dirty state tracking for preset tabs.
 */
import { ref, type Ref } from 'vue'
import type { MessageWriter } from '../../shared/notices/messages'
import type { Presets, Settings } from './core'
import { WORDS as words } from './words'

/** State tracking write flight and dirty settings for one preset file. */
export interface WriteFlight {
  at: string
  writing: boolean
  wanted: boolean
  flight: Promise<void> | null
  told: boolean
  readonly theirs: Set<keyof Settings>
  readonly changed: Ref<boolean>
  readonly errorMessage: Ref<string>
}

export function createWriteFlight(): WriteFlight {
  return {
    at: '',
    writing: false,
    wanted: false,
    flight: null,
    told: false,
    theirs: new Set<keyof Settings>(),
    changed: ref(false),
    errorMessage: ref(''),
  }
}

/** Sends the current settings to the file the read came out of. */
export const sendWrite = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  said: MessageWriter,
): Promise<void> => {
  said('')
  let answer
  try {
    answer = await core.write(path, settings, flight.at)
  } catch (error) {
    flight.errorMessage.value = words.unwritten
    console.error(error)
    return
  }
  if (answer.changed) {
    flight.changed.value = true
    return
  }
  const writeError = answer.error
  if (writeError) {
    flight.errorMessage.value = words.notSaved(writeError)
    said(flight.errorMessage.value, 'error')
    return
  }
  flight.errorMessage.value = ''
  flight.at = answer.at
  flight.theirs.clear()
  flight.told = false
}

const executeFlight = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  said: MessageWriter,
): Promise<void> => {
  await sendWrite(flight, path, settings, core, said)
  flight.writing = false
  const again = flight.wanted && !flight.changed.value
  flight.wanted = false
  if (again) {
    await requestWrite(flight, path, settings, core, said)
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
  said: MessageWriter,
): Promise<void> => {
  if (flight.writing) {
    flight.wanted = true
    return flight.flight ?? Promise.resolve()
  }
  flight.writing = true
  const inFlight = executeFlight(flight, path, settings, core, said)
  flight.flight = inFlight
  return inFlight
}

/** Flushes any unwritten changes or waits for writes in flight to settle. */
export const flushWrites = async (
  flight: WriteFlight,
  path: string,
  settings: Settings,
  core: Presets,
  said: MessageWriter,
): Promise<void> => {
  if (flight.theirs.size > 0 && !flight.changed.value) {
    await requestWrite(flight, path, settings, core, said)
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
  said: MessageWriter,
): Promise<boolean> => {
  await flushWrites(flight, path, settings, core, said)
  if (flight.theirs.size === 0 && !flight.changed.value) return true
  if (flight.told) return true
  flight.told = true
  return false
}
