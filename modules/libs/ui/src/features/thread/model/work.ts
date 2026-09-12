/**
 * The line that says what the agent has in hand, and the line under it that
 * says an answer is still on its way. One of each stands at a time.
 */
import { charsWord, type Turn } from '../lib/turn'
import type { Place } from '../lib/agent'

/**
 * A tool as the panel says it. A tool served with a title of its own arrives
 * with one; the rest arrive named the way a program is named.
 */
const getSpokenTool = (tool: string) => tool.replaceAll('_', ' ')

export interface WorkLine {
  /** The agent reached for a tool, and how much of it has been written. */
  readonly reach: (
    tool: string,
    about: string,
    place: Place | null | undefined,
    written: number,
  ) => void
  /** What it reached for has answered, so the line stops claiming to run. */
  readonly settle: () => void
  /** The line comes down. */
  readonly takeDown: () => void
  /** The wait under it, up whenever nothing more particular is known. */
  readonly setWaiting: (on: boolean) => void
  /** Both lines come down. */
  readonly clear: () => void
}

/**
 * The two lines of one exchange. `doing` and `wait` name them, and `places`
 * holds the source a line that says it opens one leads to.
 */
export function createWorkLine(
  put: (turn: Turn) => void,
  drop: (id: string) => void,
  places: Map<string, Place>,
  thinking: string,
  doing: string,
  wait: string,
): WorkLine {
  // What the agent has in hand, as far as anything has said. A call carrying
  // the text of a source is reported again every time more of it is written, so
  // the count is what moves while it is being written.
  let says = thinking
  let about = ''

  // Whether the line is up. It comes down when the answer begins, and a tool
  // answering does not put it back: what a tool did belongs above the answer it
  // led to, and a line raised here stands under that answer for good.
  let up = false

  const show = (state: 'arriving' | 'settled', written = 0) => {
    put({
      id: doing,
      voice: 'doing',
      text: says,
      about,
      aside: charsWord(written),
      state,
      ...(places.has(doing) ? { opens: true } : {}),
    })
    up = true
  }

  const takeDown = () => {
    drop(doing)
    places.delete(doing)
    up = false
  }

  const setWaiting = (on: boolean) => {
    if (on) put({ id: wait, voice: 'doing', text: thinking, about: '', state: 'arriving' })
    else drop(wait)
  }

  return {
    reach: (tool, named, place, written) => {
      says = getSpokenTool(tool)
      about = named
      // A call naming a run of a source's text names somewhere the line can be
      // pressed to open.
      if (place && place.span.to > place.span.from) places.set(doing, place)
      else places.delete(doing)
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
