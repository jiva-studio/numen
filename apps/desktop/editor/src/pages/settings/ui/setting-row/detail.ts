/** A setting description split into its first sentence and the rest. */
export interface DetailParts {
  readonly summary: string
  readonly more: string
}

const SENTENCE_END = /[.!?](?=\s+\S)/

/** Splits a description after its first sentence; the rest is empty for a single sentence. */
export function splitDetail(detail: string): DetailParts {
  const text = detail.trim()
  const end = SENTENCE_END.exec(text)
  if (end === null) return { summary: text, more: '' }
  return {
    summary: text.slice(0, end.index + 1),
    more: text.slice(end.index + 1).trim(),
  }
}
