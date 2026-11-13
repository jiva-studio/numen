/**
 * The status line showing the tool the agent is running, and the wait line
 * under it, shown whenever no more specific status is known.
 */
import { writeCharCount } from '../lib/turn'
import type { Turn } from '../../types'

const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

/** What the line says about one tool call: its title, what it is about, and the file it works on. */
export interface WorkCall {
  readonly title: string
  readonly subject: string
  readonly path: string
}

/** write shows how many characters of the text of the current call have arrived, until it settles. */
export interface WorkLine {
  readonly reach: (call: WorkCall) => void
  readonly write: (count: number) => void
  readonly settle: () => void
  readonly dropWorkLine: () => void
  readonly setWaiting: (isOn: boolean) => void
  readonly clear: () => void
}

/** paths holds the file a line opens when pressed, by the id of the line. */
export interface WorkLineOptions {
  readonly put: (turn: Turn) => void
  readonly drop: (id: string) => void
  readonly paths: Map<string, string>
  readonly waitWords: string
}

/**
 * The two lines of one exchange. `workId` and `wait` are their identifiers, and
 * `paths` holds the file a line opens when pressed.
 */
export function createWorkLine(options: WorkLineOptions, workId: string, wait: string): WorkLine {
  const { put, drop, paths, waitWords } = options
  let says = waitWords
  let subject = ''
  let written = 0

  // Whether the line is up. It comes down when the answer begins, and a tool
  // answering does not put it back: what a tool did belongs above the answer it
  // led to, and a line raised here is placed under that answer for good.
  let up = false

  const show = (state: 'arriving' | 'settled') => {
    put({
      id: workId,
      voice: 'doing',
      text: says,
      subject,
      state,
      ...(written > 0 ? { aside: writeCharCount(written) } : {}),
      ...(paths.has(workId) ? { canOpen: true } : {}),
    })
    up = true
  }

  const dropWorkLine = () => {
    drop(workId)
    paths.delete(workId)
    up = false
  }

  const setWaiting = (on: boolean) => {
    if (on) put({ id: wait, voice: 'doing', text: waitWords, subject: '', state: 'arriving' })
    else drop(wait)
  }

  return {
    reach: (call) => {
      says = getSpokenTool(call.title)
      subject = call.subject
      written = 0
      if (call.path) paths.set(workId, call.path)
      else paths.delete(workId)
      show('arriving')
    },
    write: (count) => {
      written = count
      if (up) show('arriving')
    },
    settle: () => {
      written = 0
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
