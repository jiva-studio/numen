/**
 * The line that says what the agent has in hand, and the line under it that
 */
import { writeCharCount, type Turn } from '../lib/turn'
import type { SourceLocation } from '@numen/plugin-sdk'

/**
 * A tool as the panel says it. A tool served with a title of its own arrives
 * with one; the rest arrive named the way a program is named.
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
 * The two lines of one exchange. `workId` and `wait` name them, and `locations`
 * holds the source a line that says it opens one leads to.
 */
export function createWorkLine(options: WorkLineOptions, workId: string, wait: string): WorkLine {
  const { put, drop, locations, waitWords } = options
  // What the agent has in hand, as far as anything has said. A call carrying
  // the text of a source is reported again every time more of it is written, so
  // the count is what moves while it is being written.
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
