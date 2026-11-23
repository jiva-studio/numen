/**
 * The status lines of one exchange: one line for each tool call, kept in the
 * order the calls were made, and the wait line, shown whenever no more specific
 * status is known.
 */
import { writeCharCount } from '../lib/turn'
import type { Turn } from '../../types'

const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

/** What the line says about one tool call: its title, what it is about, and the file it works on. */
interface WorkCall {
  readonly title: string
  readonly subject: string
  readonly path: string
}

/** write shows how many characters of the text of a call have arrived, until the call ends. */
export interface WorkLine {
  readonly reach: (callId: string, call: WorkCall) => void
  readonly write: (callId: string, count: number) => void
  readonly end: (callId: string) => void
  readonly setWaiting: (isOn: boolean) => void
  readonly clear: () => void
}

/** paths holds the file a line opens when pressed, by the id of the line. */
export interface WorkLineOptions {
  readonly put: (turn: Turn) => void
  readonly drop: (id: string) => void
  readonly paths: Map<string, string>
  readonly waitWords: string
  readonly createId: () => string
}

interface CallLine {
  readonly id: string
  readonly says: string
  readonly subject: string
  written: number
  isEnded: boolean
}

/** The lines of one exchange, under `wait` for the wait line and names taken from `nextId` for each call. */
export function createWorkLine(options: WorkLineOptions, wait: string): WorkLine {
  const { put, drop, paths, waitWords, createId: nextId } = options
  const lines = new Map<string, CallLine>()

  const show = (line: CallLine) => {
    put({
      id: line.id,
      voice: 'doing',
      text: line.says,
      subject: line.subject,
      state: line.isEnded ? 'settled' : 'arriving',
      ...(line.written > 0 && !line.isEnded ? { aside: writeCharCount(line.written) } : {}),
      ...(paths.has(line.id) ? { canOpen: true } : {}),
    })
  }

  return {
    reach: (callId, call) => {
      const line: CallLine = {
        id: lines.get(callId)?.id ?? nextId(),
        says: getSpokenTool(call.title),
        subject: call.subject,
        written: 0,
        isEnded: false,
      }
      lines.set(callId, line)
      if (call.path) paths.set(line.id, call.path)
      else paths.delete(line.id)
      show(line)
    },
    write: (callId, count) => {
      const line = lines.get(callId)
      if (!line || line.isEnded) return
      line.written = count
      show(line)
    },
    end: (callId) => {
      const line = lines.get(callId)
      if (!line || line.isEnded) return
      line.isEnded = true
      show(line)
    },
    setWaiting: (on) => {
      if (on) put({ id: wait, voice: 'doing', text: waitWords, subject: '', state: 'arriving' })
      else drop(wait)
    },
    clear: () => {
      for (const line of lines.values()) {
        if (line.isEnded) continue
        line.isEnded = true
        show(line)
      }
      drop(wait)
    },
  }
}
