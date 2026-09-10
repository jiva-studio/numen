/**
 * Write queuing, flight synchronization, and dirty state tracking for preset tabs.
 */
import { ref, type Ref } from 'vue'
import type { MessageWriter } from '../shared/notices/messages'
import type { Presets, Settings } from './core'
import { WORDS as words } from './words'

/** State tracking write flight and dirty settings for one preset file. */
export interface WriteFlight {
  at: string
  isWriting: boolean
  writing: boolean
  isSelected: boolean
  wanted: boolean
  flight: Promise<void> | null
  hasMessage: boolean
  told: boolean
  readonly theirs: Set<keyof Settings>
  readonly changed: Ref<boolean>
  readonly saying: Ref<string>
}

export function createWriteFlight(): WriteFlight {
  return {
    at: '',
    isWriting: false,
    writing: false,
    isSelected: false,
    wanted: false,
    flight: null,
    hasMessage: false,
    told: false,
    theirs: new Set<keyof Settings>(),
    changed: ref(false),
    saying: ref(''),
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
    flight.saying.value = words.unwritten
    console.error(error)
    return
  }
  if (answer.changed) {
    flight.changed.value = true
    return
  }
  if (answer.refusal) {
    flight.saying.value = words.notSaved(answer.refusal)
    said(flight.saying.value, 'refusal')
    return
  }
  flight.saying.value = ''
  flight.at = answer.at
  flight.theirs.clear()
  flight.hasMessage = false
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
  flight.isWriting = false
  flight.writing = false
  const again = (flight.isSelected || flight.wanted) && !flight.changed.value
  flight.isSelected = false
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
  if (flight.isWriting || flight.writing) {
    flight.isSelected = true
    flight.wanted = true
    return flight.flight ?? Promise.resolve()
  }
  flight.isWriting = true
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
  if (flight.hasMessage || flight.told) return true
  flight.hasMessage = true
  flight.told = true
  return false
}
