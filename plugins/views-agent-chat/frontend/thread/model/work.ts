/**
 * The status line showing the tool the agent is running, and the wait line
 * under it, shown whenever no more specific status is known.
 */
import { writeCharCount } from '../lib/turn'
import type { Turn } from '../../types'
import type { SourceLocation } from '@numen/plugin-sdk'

const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

interface WorkCall {
  readonly tool: string
  readonly subject: string
  readonly location?: SourceLocation | null | undefined
  readonly written: number
}

export interface WorkLine {
  readonly reach: (call: WorkCall) => void
  readonly settle: () => void
  readonly dropWorkLine: () => void
  readonly setWaiting: (isOn: boolean) => void
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

  const dropWorkLine = () => {
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
    dropWorkLine,
    setWaiting,
    clear: () => {
      dropWorkLine()
      setWaiting(false)
    },
  }
}
