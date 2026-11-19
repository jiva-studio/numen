/**
 * SQL and JSON syntax tokenizer for structured log properties.
 */
import type { SyntaxToken } from '../types'

const SQL_KEYWORDS = new Set([
  'SELECT',
  'FROM',
  'WHERE',
  'INNER',
  'LEFT',
  'RIGHT',
  'OUTER',
  'CROSS',
  'JOIN',
  'ON',
  'AND',
  'OR',
  'NOT',
  'IN',
  'IS',
  'NULL',
  'AS',
  'ORDER',
  'GROUP',
  'BY',
  'HAVING',
  'LIMIT',
  'OFFSET',
  'INSERT',
  'INTO',
  'VALUES',
  'UPDATE',
  'SET',
  'DELETE',
  'CONFLICT',
  'DO',
  'NOTHING',
  'RETURNING',
  'DISTINCT',
  'COUNT',
  'MIN',
  'MAX',
  'SUM',
  'COALESCE',
  'CASE',
  'WHEN',
  'THEN',
  'ELSE',
  'END',
  'ASC',
  'DESC',
  'PRAGMA',
  'BEGIN',
  'COMMIT',
  'ROLLBACK',
  'CREATE',
  'DROP',
  'ALTER',
  'TABLE',
  'INDEX',
  'IF',
  'EXISTS',
  'PRIMARY',
  'KEY',
  'FOREIGN',
  'REFERENCES',
  'WITH',
  'RECURSIVE',
  'UNION',
  'ALL',
  'LIKE',
  'GLOB',
  'MATCH',
  'BETWEEN',
])

const SQL_TOKEN_PATTERN =
  /(--[^\n]*|'(?:''|[^'])*'|"(?:""|[^"])*"|\b\d+(?:\.\d+)?\b|\b[A-Za-z_][A-Za-z0-9_]*\b|[(),;=<>!?*+-]+)/g

const JSON_TOKEN_PATTERN =
  /("(?:\\.|[^"\\])*"(?=\s*:)|"(?:\\.|[^"\\])*"|\b(?:true|false|null)\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|[{}[\]:,])/g

function appendToken(out: SyntaxToken[], kind: SyntaxToken['kind'], text: string): void {
  if (!text) return
  const prev = out[out.length - 1]
  if (prev && prev.kind === kind) {
    out[out.length - 1] = { kind, text: prev.text + text }
    return
  }
  out.push({ kind, text })
}

function tokenizeWithPattern(
  text: string,
  pattern: RegExp,
  classify: (match: string, after: string) => SyntaxToken['kind'],
): readonly SyntaxToken[] {
  const out: SyntaxToken[] = []
  let lastIndex = 0
  for (const match of text.matchAll(pattern)) {
    const index = match.index ?? 0
    if (index > lastIndex) {
      appendToken(out, 'plain', text.slice(lastIndex, index))
    }
    const tokenText = match[0]
    const after = text.slice(index + tokenText.length)
    appendToken(out, classify(tokenText, after), tokenText)
    lastIndex = index + tokenText.length
  }
  if (lastIndex < text.length) {
    appendToken(out, 'plain', text.slice(lastIndex))
  }
  return out
}

function classifySqlToken(token: string): SyntaxToken['kind'] {
  if (token.startsWith('--')) {
    return 'comment'
  }
  if (token.startsWith("'") || token.startsWith('"')) {
    return 'string'
  }
  if (/^\d/.test(token)) {
    return 'number'
  }
  if (/^[(),;=<>!?*+-]+$/.test(token)) {
    return token === '?' ? 'type' : 'punctuation'
  }
  return SQL_KEYWORDS.has(token.toUpperCase()) ? 'keyword' : 'plain'
}

function classifyJsonToken(token: string, after: string): SyntaxToken['kind'] {
  if (token.startsWith('"')) {
    return /^\s*:/.test(after) ? 'key' : 'string'
  }
  if (token === 'true' || token === 'false' || token === 'null') {
    return 'type'
  }
  return /^-?\d/.test(token) ? 'number' : 'punctuation'
}

/** Splits formatted SQL text into syntax tokens. */
export function tokenizeSql(text: string): readonly SyntaxToken[] {
  return tokenizeWithPattern(text, SQL_TOKEN_PATTERN, classifySqlToken)
}

/** Splits formatted JSON text into syntax tokens. */
export function tokenizeJson(text: string): readonly SyntaxToken[] {
  return tokenizeWithPattern(text, JSON_TOKEN_PATTERN, classifyJsonToken)
}
