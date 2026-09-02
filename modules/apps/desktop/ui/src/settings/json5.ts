/**
 * JSON5 read and written.
 *
 * A person editing a setting by hand writes comments beside it, leaves a comma
 * after the last member, and quotes a key or does not. This reads all of that
 * and hands back what JSON holds; what is written back is JSON, which is what
 * the settings file is.
 *
 * The subset read is the one a settings file is written in: comments, trailing
 * commas, single quotes and bare keys. A number written the way JSON5 allows
 * and JSON does not is left as it was typed, and refused.
 */

/** What is not JSON5 at all. */
export class Unreadable extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'Unreadable'
  }
}

/** A key that needs no quotes: what an identifier is written with. */
const BARE = /[A-Za-z_$][A-Za-z0-9_$]*/y

/** The text with the comments taken out and everything JSON holds left in. */
export const read = (text: string): unknown => JSON.parse(plain(text))

/** A value written back into the settings, which are JSON. */
export const write = (value: unknown): string => JSON.stringify(value, null, 2)

/**
 * The same text as JSON: the comments gone, the last comma of a list or an
 * object gone, single quotes turned round, and a bare key quoted.
 */
export const plain = (text: string): string => {
  let said = ''
  let at = 0
  while (at < text.length) {
    const one = text[at]!
    if (one === '/' && (text[at + 1] === '/' || text[at + 1] === '*')) {
      at = past(text, at)
      continue
    }
    if (one === '"' || one === "'") {
      const [written, next] = quoted(text, at)
      said += written
      at = next
      continue
    }
    // A string is written out whole, so a comma at the end of what has been
    // written is a comma between members, and one before a close is a comma
    // with nothing behind it.
    if (one === '}' || one === ']') {
      const kept = said.trimEnd()
      said = kept.endsWith(',') ? kept.slice(0, -1) : said
      said += one
      at += 1
      continue
    }
    BARE.lastIndex = at
    const bare = BARE.exec(text)
    if (bare) {
      said += key(text, BARE.lastIndex) ? JSON.stringify(bare[0]) : bare[0]
      at = BARE.lastIndex
      continue
    }
    said += one
    at += 1
  }
  return said
}

/** Where the text takes up again past a comment. */
const past = (text: string, at: number): number => {
  if (text[at + 1] === '/') {
    const end = text.indexOf('\n', at)
    return end < 0 ? text.length : end
  }
  const end = text.indexOf('*/', at + 2)
  if (end < 0) throw new Unreadable('a comment nothing closes')
  return end + 2
}

/** Whether what follows a word is the colon that makes it a key. */
const key = (text: string, at: number): boolean => {
  while (at < text.length && ' \t\r\n'.includes(text[at]!)) at += 1
  return text[at] === ':'
}

/** The letters that stand for a character that cannot be typed. */
const ESCAPES: Readonly<Record<string, string>> = {
  b: '\b',
  f: '\f',
  n: '\n',
  r: '\r',
  t: '\t',
  v: '\v',
  '0': '\0',
}

/** A character named by its number: how many digits it takes, and what they are. */
const NAMED: Readonly<Record<string, { length: number; digits: RegExp }>> = {
  u: { length: 4, digits: /^[0-9a-fA-F]{4}$/ },
  x: { length: 2, digits: /^[0-9a-fA-F]{2}$/ },
}

/** One string as it reads, written the way JSON writes it, and where the text goes on. */
const quoted = (text: string, at: number): [string, number] => {
  const mark = text[at]
  let said = ''
  let walk = at + 1
  while (walk < text.length) {
    const one = text[walk]!
    if (one === '\\') {
      const next = text[walk + 1]
      if (next === undefined) break
      // A line broken with a backslash is one line.
      if (next === '\n') {
        walk += 2
        continue
      }
      const named = NAMED[next]
      if (named) {
        const code = text.slice(walk + 2, walk + 2 + named.length)
        if (!named.digits.test(code)) throw new Unreadable('a character nothing names')
        said += String.fromCharCode(parseInt(code, 16))
        walk += 2 + named.length
        continue
      }
      said += ESCAPES[next] ?? next
      walk += 2
      continue
    }
    if (one === mark) return [JSON.stringify(said), walk + 1]
    said += one
    walk += 1
  }
  throw new Unreadable('a string nothing closes')
}
