/**
 * Frontmatter metadata parser for Markdown notes.
 */
export interface FrontmatterResult {
  readonly data: Readonly<Record<string, unknown>>
  readonly content: string
}

function parseFrontmatterValue(raw: string): unknown {
  const trimmed = raw.trim()
  if (trimmed === 'true') return true
  if (trimmed === 'false') return false
  if (trimmed === 'null') return null
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) return Number(trimmed)
  if (
    (trimmed.startsWith('"') && trimmed.endsWith('"')) ||
    (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1)
  }
  return trimmed
}

export function parseFrontmatter(text: string): FrontmatterResult {
  if (!text.startsWith('---\n') && !text.startsWith('---\r\n')) {
    return { data: {}, content: text }
  }

  const newlineOffset = text.indexOf('\n')
  const rest = text.slice(newlineOffset + 1)
  const delimiterMatch = rest.match(/\n---\s*(\r?\n|$)/)
  if (!delimiterMatch || delimiterMatch.index === undefined) {
    return { data: {}, content: text }
  }

  const rawYaml = rest.slice(0, delimiterMatch.index)
  const content = rest.slice(delimiterMatch.index + delimiterMatch[0].length)
  const data: Record<string, unknown> = {}

  for (const line of rawYaml.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const colonIndex = trimmed.indexOf(':')
    if (colonIndex > 0) {
      const key = trimmed.slice(0, colonIndex).trim()
      const rawValue = trimmed.slice(colonIndex + 1).trim()
      data[key] = parseFrontmatterValue(rawValue)
    }
  }

  return { data, content }
}
