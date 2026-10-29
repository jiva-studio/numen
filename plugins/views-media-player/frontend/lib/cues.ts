/**
 * Maps the edited transcript text back to the millisecond spans of its cues.
 *
 * One cue is one line, so an edit to the text is an edit to the lines and the
 * times follow it. An edited line keeps its times; joined lines take the first
 * line's start and the last line's end; a split line divides its span in
 * proportion to the characters on each side of the split.
 */

/** One cue: its text and the milliseconds it covers. */
export interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

/**
 * The index of the cue active at a millisecond, or the last cue before it when
 * a gap contains the millisecond. Returns -1 before the first cue begins.
 */
export const findCueAt = (cues: readonly Cue[], ms: number): number => {
  for (let index = cues.length - 1; index >= 0; index--) {
    if (cues[index]!.from <= ms) return index
  }
  return -1
}

/** The prose of a transcript: one cue to a line. */
export const getText = (cues: readonly Cue[]): string => cues.map((cue) => cue.text).join('\n')

/** Whether two runs of cues say the same words at the same moments. */
export const isSame = (a: readonly Cue[], b: readonly Cue[]): boolean =>
  a.length === b.length &&
  a.every(
    (cue, index) =>
      cue.text === b[index]!.text && cue.from === b[index]!.from && cue.to === b[index]!.to,
  )

/**
 * Where the lines of `text` fall in the cues they were edited from: one cue
 * for each line, in the order they are read, and an emptied line as the empty
 * span it now covers.
 */
/**
 * How many lines at the start and at the end of the text are unchanged. The
 * lines between them are the edited ones.
 */
const countSame = (
  lines: readonly string[],
  was: readonly Cue[],
): { head: number; tail: number } => {
  let head = 0
  // Stryker disable next-line ConditionalExpression: out of bounds lines[head] evaluates to undefined and halts loop.
  while (head < lines.length && head < was.length && lines[head] === was[head]!.text) head++
  let tail = 0
  while (
    tail < lines.length - head &&
    tail < was.length - head &&
    lines[lines.length - 1 - tail] === was[was.length - 1 - tail]!.text
  ) {
    tail++
  }
  return { head, tail }
}

export const spanCues = (was: readonly Cue[], text: string): readonly Cue[] => {
  const lines = text.split('\n')
  const { head, tail } = countSame(lines, was)

  const headCues = was.slice(0, head)
  const rest = was.slice(was.length - tail)
  const old = was.slice(head, was.length - tail)
  const newLines = lines.slice(head, lines.length - tail)

  // Lines inserted where no cue existed take an empty span at the boundary where
  // they were inserted.
  const start = old.length
    ? old[0]!.from
    : (headCues[headCues.length - 1]?.to ?? rest[0]?.from ?? 0)
  const end = old.length ? old[old.length - 1]!.to : start

  const middle =
    old.length === newLines.length
      ? newLines.map((one, index) => ({ text: one, from: old[index]!.from, to: old[index]!.to }))
      : spread(newLines, start, end)

  return [...headCues, ...middle, ...rest]
}

/**
 * The transcript as it reads after the edit, ready to be saved. A line emptied
 * entirely is dropped.
 */
export const applyCues = (was: readonly Cue[], text: string): readonly Cue[] =>
  spanCues(was, text).filter((cue) => cue.text !== '')

/**
 * Lines laid over one span, each taking of it what its characters are of all
 * of them. Lines carrying no characters at all share the span's start.
 */
const spread = (texts: readonly string[], start: number, end: number): Cue[] => {
  const all = texts.reduce((sum, one) => sum + one.length, 0)
  const cues: Cue[] = []
  let from = start
  let charCount = 0
  for (let index = 0; index < texts.length; index++) {
    const text = texts[index]!
    charCount += text.length
    const last = index === texts.length - 1
    let to = end
    if (!last) {
      to =
        all === 0
          ? from
          : Math.min(end, Math.max(from, start + Math.round(((end - start) * charCount) / all)))
    }
    cues.push({ text, from, to })
    from = to
  }
  return cues
}
