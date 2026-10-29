/**
 * The status line showing the tool the agent is running, and the wait line
 * under it, shown whenever no more specific status is known.
 */
import { writeCharCount, type Turn } from '../lib/turn'
import type { SourceLocation } from '@numen/plugin-sdk'

/**
 * A tool name as displayed in the panel. A tool with its own title arrives with
 * it; the others arrive named as a program is named.
 */
const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

export interface WorkCall {
  readonly tool: string
  readonly subject: string
  readonly location?: SourceLocation | null | undefined
  readonly written: number
}

export interface WorkLine {
  /** The agent reached for a tool, and how much of it has been written. */
  readonly reach: (call: WorkCall) => void
  /** What it reached for has answered, so the line stops claiming to run. */
  readonly settle: () => void
  /** The line comes down. */
  readonly takeDown: () => void
  /** The wait under it, up whenever nothing more particular is known. */
  readonly setWaiting: (isOn: boolean) => void
  /** Both lines come down. */
  readonly clear: () => void
}

export interface WorkLineOptions {
  readonly put: (turn: Turn) => void
  readonly drop: (id: string) => void
  readonly locations: Map<string, SourceLocation>
  readonly waitWords: string
}

/**
 * The two lines of one exchange. `workId` and `wait` are their identifiers, and
 * `locations` holds the source location a line opens when pressed.
 */
export function createWorkLine(options: WorkLineOptions, workId: string, wait: string): WorkLine {
  const { put, drop, locations, waitWords } = options
  // The status text for the current call. A call carrying the text of a source
  // is reported again each time more of it is written, so the written count is
  // what changes while it is being written.
  let says = waitWords
  let subject: string

  // Whether the line is up. It comes down when the answer begins, and a tool
  // answering does not put it back: what a tool did belongs above the answer it
  // led to, and a line raised here is placed under that answer for good.
  let up = false

  const show = (state: 'arriving' | 'settled', count = 0) => {
    put({
      id: workId,
      voice: 'doing',
      text: says,
      subject,
      aside: writeCharCount(count),
      state,
      ...(locations.has(workId) ? { canOpen: true } : {}),
    })
    up = true
  }

  const takeDown = () => {
    drop(workId)
    locations.delete(workId)
    up = false
  }

  const setWaiting = (on: boolean) => {
    if (on) put({ id: wait, voice: 'doing', text: waitWords, subject: '', state: 'arriving' })
    else drop(wait)
  }

  return {
    reach: (call) => {
      says = getSpokenTool(call.tool)
      subject = call.subject
      // A call naming a run of a source's text names somewhere the line can be
      // pressed to open.
      if (call.location && call.location.range.end > call.location.range.start) {
        locations.set(workId, call.location)
      } else {
        locations.delete(workId)
      }
      show('arriving', call.written)
    },
    settle: () => {
      if (up) show('settled')
    },
    takeDown,
    setWaiting,
    clear: () => {
      takeDown()
      setWaiting(false)
    },
  }
}
