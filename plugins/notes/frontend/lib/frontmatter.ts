/**
 * Frontmatter metadata parser for Markdown notes.
 */
export interface FrontmatterResult {
  readonly data: Readonly<Record<string, unknown>>
  readonly content: string
}

function parseFrontmatterValue(raw: string): unknown {
  const value = raw.trim()
  if (value === 'true') return true
  if (value === 'false') return false
  if (value === 'null') return null
  if (/^-?\d+(\.\d+)?$/.test(value)) return Number(value)
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1)
  }
  return value
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
    const cleanLine = line.trim()
    if (!cleanLine || cleanLine.startsWith('#')) continue
    const colonIndex = cleanLine.indexOf(':')
    if (colonIndex > 0) {
      const key = cleanLine.slice(0, colonIndex).trim()
      const rawValue = cleanLine.slice(colonIndex + 1).trim()
      data[key] = parseFrontmatterValue(rawValue)
    }
  }

  return { data, content }
}
