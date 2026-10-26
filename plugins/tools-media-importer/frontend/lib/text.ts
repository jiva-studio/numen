/**
 * How a link's text and a moment of its copy are written out.
 */

/** The paragraphs of a text: its lines that hold anything, each trimmed. */
export function splitParagraphs(text: string): readonly string[] {
  return text
    .split(/\n+/)
    .map((line) => line.trim())
    .filter((line) => line !== '')
}

/** A moment, in milliseconds, as minutes and seconds: `1:07`, and hours before them past an hour. */
export function formatClock(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = String(total % 60).padStart(2, '0')
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, '0')}:${seconds}`
    : `${minutes}:${seconds}`
}
