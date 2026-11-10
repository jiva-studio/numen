/**
 * Finds frontmatter a YAML reader would refuse, without reading the YAML.
 */
interface DelimiterState {
  inSingleQuote: boolean
  inDoubleQuote: boolean
  bracketDepth: number
  braceDepth: number
}

function scanQuoteChar(
  char: string,
  nextChar: string | undefined,
  state: DelimiterState,
): { skipNext: boolean; isHandled: boolean } {
  if (state.inSingleQuote) {
    if (char === "'") {
      if (nextChar === "'") return { skipNext: true, isHandled: true }
      state.inSingleQuote = false
    }
    return { skipNext: false, isHandled: true }
  }
  if (state.inDoubleQuote) {
    if (char === '\\') return { skipNext: true, isHandled: true }
    if (char === '"') state.inDoubleQuote = false
    return { skipNext: false, isHandled: true }
  }
  return { skipNext: false, isHandled: false }
}

function updateBracketDepth(char: string, state: DelimiterState): void {
  if (char === '[') state.bracketDepth++
  else if (char === ']') state.bracketDepth--
  else if (char === '{') state.braceDepth++
  else if (char === '}') state.braceDepth--
}

function scanCharDelimiters(
  char: string,
  nextChar: string | undefined,
  state: DelimiterState,
): { skipNext: boolean; isDone: boolean } {
  const quote = scanQuoteChar(char, nextChar, state)
  if (quote.isHandled) return { skipNext: quote.skipNext, isDone: false }

  if (char === '#') return { skipNext: false, isDone: true }
  if (char === "'") state.inSingleQuote = true
  else if (char === '"') state.inDoubleQuote = true
  else updateBracketDepth(char, state)

  return { skipNext: false, isDone: false }
}

function scanLineDelimiters(line: string, state: DelimiterState): boolean {
  for (let i = 0; i < line.length; i++) {
    const res = scanCharDelimiters(line[i]!, line[i + 1], state)
    if (res.skipNext) i++
    if (res.isDone) break
    if (state.bracketDepth < 0 || state.braceDepth < 0) return true
  }
  return false
}

function isTopLevelKeyLine(line: string, lineContent: string): boolean {
  return (
    !line.startsWith(' ') &&
    !line.startsWith('\t') &&
    !lineContent.startsWith('#') &&
    !lineContent.startsWith('-')
  )
}

function checkTopLevelMappingLine(lineContent: string, topKeys: Set<string>): boolean {
  const colonIdx = lineContent.indexOf(':')
  if (colonIdx <= 0) return true
  const key = lineContent.slice(0, colonIdx).trim()
  if (topKeys.has(key)) return true
  topKeys.add(key)
  const after = lineContent.slice(colonIdx + 1).trim()
  return after.endsWith(':') && !after.startsWith('{') && !after.startsWith('[')
}

function isTopLevel(state: DelimiterState): boolean {
  return (
    state.bracketDepth === 0 &&
    state.braceDepth === 0 &&
    !state.inSingleQuote &&
    !state.inDoubleQuote
  )
}

function isMalformedLine(
  line: string,
  lineContent: string,
  state: DelimiterState,
  topKeys: Set<string>,
): boolean {
  if (
    isTopLevel(state) &&
    isTopLevelKeyLine(line, lineContent) &&
    checkTopLevelMappingLine(lineContent, topKeys)
  ) {
    return true
  }
  return scanLineDelimiters(line, state)
}

function hasMalformedFrontmatter(rawYaml: string): boolean {
  const state: DelimiterState = {
    inSingleQuote: false,
    inDoubleQuote: false,
    bracketDepth: 0,
    braceDepth: 0,
  }
  const topKeys = new Set<string>()

  for (const line of rawYaml.split(/\r?\n/)) {
    const lineContent = line.trim()
    if (lineContent && isMalformedLine(line, lineContent, state, topKeys)) return true
  }

  return (
    state.inSingleQuote || state.inDoubleQuote || state.bracketDepth !== 0 || state.braceDepth !== 0
  )
}

/** The words for frontmatter that cannot be read; empty where it can be, or there is none. */
export function getFrontmatterError(content: string): string {
  if (!content.startsWith('---\n') && !content.startsWith('---\r\n')) {
    return ''
  }
  const newlineOffset = content.indexOf('\n')
  const rest = content.slice(newlineOffset + 1)
  const delimiterMatch = rest.match(/\n---\s*(\r?\n|$)/)
  if (!delimiterMatch || delimiterMatch.index === undefined) {
    return ''
  }
  const rawYaml = rest.slice(0, delimiterMatch.index)
  if (hasMalformedFrontmatter(rawYaml)) {
    return 'frontmatter unreadable'
  }
  return ''
}
