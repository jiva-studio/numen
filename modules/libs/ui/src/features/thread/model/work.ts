/**
 * The line that says what the agent has in hand, and the line under it that
 * says an answer is still on its way. One of each stands at a time.
 */
import { writeCharCount, type Turn } from '../lib/turn'
import type { SourceLocation } from '../lib/agent'

/**
 * A tool as the panel says it. A tool served with a title of its own arrives
 * with one; the rest arrive named the way a program is named.
 */
const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

export interface WorkLine {
  /** The agent reached for a tool, and how much of it has been written. */
  readonly reach: (
    tool: string,
    subject: string,
    location: SourceLocation | null | undefined,
    written: number,
  ) => void
  /** What it reached for has answered, so the line stops claiming to run. */
  readonly settle: () => void
  /** The line comes down. */
  readonly takeDown: () => void
  /** The wait under it, up whenever nothing more particular is known. */
  readonly setWaiting: (isOn: boolean) => void
  /** Both lines come down. */
  readonly clear: () => void
}

/**
 * The two lines of one exchange. `workId` and `wait` name them, and `locations`
 * holds the source a line that says it opens one leads to.
 */
export function createWorkLine(
  put: (turn: Turn) => void,
  drop: (id: string) => void,
  locations: Map<string, SourceLocation>,
  waitWords: string,
  workId: string,
  wait: string,
): WorkLine {
  // What the agent has in hand, as far as anything has said. A call carrying
  // the text of a source is reported again every time more of it is written, so
  // the count is what moves while it is being written.
  let says = waitWords
  let subject = ''

  // Whether the line is up. It comes down when the answer begins, and a tool
  // answering does not put it back: what a tool did belongs above the answer it
  // led to, and a line raised here stands under that answer for good.
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
    reach: (tool, named, location, written) => {
      says = getSpokenTool(tool)
      subject = named
      // A call naming a run of a source's text names somewhere the line can be
      // pressed to open.
      if (location && location.span.to > location.span.from) locations.set(workId, location)
      else locations.delete(workId)
      show('arriving', written)
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
