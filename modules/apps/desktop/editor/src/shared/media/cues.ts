/**
 * The words as they now read, back against the milliseconds they were said in.
 *
 * One cue is one line, so an edit to the words is an edit to the lines and the
 * times follow it. A line that only changed keeps its own times; lines joined
 * take the first one's start and the last one's end; a line split shares its
 * span out where the split fell in its characters.
 */

/** One span of speech: what was said, and the milliseconds it covers. */
export interface Cue {
  readonly text: string
  readonly from: number
  readonly to: number
}

/** The prose of a transcript: one cue to a line. */
export const spoken = (cues: readonly Cue[]): string => cues.map((cue) => cue.text).join('\n')

/** Whether two runs of cues say the same words at the same moments. */
export const same = (a: readonly Cue[], b: readonly Cue[]): boolean =>
  a.length === b.length &&
  a.every(
    (cue, at) => cue.text === b[at]!.text && cue.from === b[at]!.from && cue.to === b[at]!.to,
  )

/**
 * Where the lines of `text` fall in the cues they were edited from: one cue
 * for each line, in the order they are read, and an emptied line as the empty
 * span it now covers.
 */
export const spanning = (was: readonly Cue[], text: string): readonly Cue[] => {
  const lines = text.split('\n')

  // The lines that read as they did, from either end. What is left between
  // them is what the person changed.
  let head = 0
  while (head < lines.length && head < was.length && lines[head] === was[head]!.text) head++
  let tail = 0
  while (
    tail < lines.length - head &&
    tail < was.length - head &&
    lines[lines.length - 1 - tail] === was[was.length - 1 - tail]!.text
  ) {
    tail++
  }

  const kept = was.slice(0, head)
  const rest = was.slice(was.length - tail)
  const old = was.slice(head, was.length - tail)
  const fresh = lines.slice(head, lines.length - tail)

  // Lines typed where no cue stood take an empty span at the boundary they
  // were typed on.
  const start = old.length ? old[0]!.from : (kept[kept.length - 1]?.to ?? rest[0]?.from ?? 0)
  const end = old.length ? old[old.length - 1]!.to : start

  const middle =
    old.length === fresh.length
      ? fresh.map((one, at) => ({ text: one, from: old[at]!.from, to: old[at]!.to }))
      : spread(fresh, start, end)

  return [...kept, ...middle, ...rest]
}

/**
 * The transcript as it now reads, for the application to keep. A line emptied
 * entirely is dropped.
 */
export const cued = (was: readonly Cue[], text: string): readonly Cue[] =>
  spanning(was, text).filter((cue) => cue.text !== '')

/**
 * Lines laid over one span, each taking of it what its characters are of all
 * of them. Lines carrying no characters at all share the span's start.
 */
const spread = (texts: readonly string[], start: number, end: number): Cue[] => {
  const all = texts.reduce((sum, one) => sum + one.length, 0)
  const cues: Cue[] = []
  let from = start
  let seen = 0
  for (let at = 0; at < texts.length; at++) {
    const text = texts[at]!
    seen += text.length
    const last = at === texts.length - 1
    const to = last
      ? end
      : all === 0
        ? from
        : Math.min(end, Math.max(from, start + Math.round(((end - start) * seen) / all)))
    cues.push({ text, from, to })
    from = to
  }
  return cues
}
