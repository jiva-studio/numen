/**
 * A catch that throws the error away and says nothing about why.
 *
 * A swallowed failure is the one fault class no test and no story sees: the
 * work does not happen, nothing is drawn, nothing is said, and the screen sits
 * there looking exactly as it did before. The window has a voice, and every
 * catch that reaches a person goes through it. What is left over is the catch
 * that reaches nobody, and nothing but a person writing it down tells a
 * deliberate one from a forgotten one.
 *
 * So the invariant: a catch either does something with the error it was handed
 * or carries a comment saying why there was nothing to do. A bare `catch {}`
 * is neither, and is never self-evidently deliberate.
 */
import { blocks } from './source.mjs'

/** The parts of a file that are code: a component's is in its script blocks. */
export const code = (at, text) => (at.endsWith('.vue') ? blocks(text, 'script') : [text])

/**
 * `source` with every comment and every string's text blanked out, character
 * for character, so what is left is code standing at the offsets it had.
 *
 * The `${…}` of a template literal is code and is kept. A regular literal is
 * left alone: telling one from a division needs the parse this file exists to
 * do without, and no brace inside one has ever changed what a catch spans.
 */
export const masked = (source) => scan(source, 0, false).out

/** The same, from `from`, stopping at the `}` that closes a `${` where `nested`. */
function scan(source, from, nested) {
  let out = ''
  let i = from
  let depth = 0
  const blank = (to) => {
    for (; i < to && i < source.length; i += 1) out += source[i] === '\n' ? '\n' : ' '
  }

  while (i < source.length) {
    const two = source.slice(i, i + 2)
    if (two === '//') {
      const ends = source.indexOf('\n', i)
      blank(ends < 0 ? source.length : ends)
      continue
    }
    if (two === '/*') {
      const ends = source.indexOf('*/', i + 2)
      blank(ends < 0 ? source.length : ends + 2)
      continue
    }

    const one = source[i]
    if (one === '}' && nested && depth === 0) return { out, next: i }
    if (one === '{') depth += 1
    if (one === '}') depth -= 1

    if (one === "'" || one === '"') {
      out += one
      i += 1
      let ends = i
      for (; ends < source.length; ends += 1) {
        if (source[ends] === '\\') ends += 1
        else if (source[ends] === one || source[ends] === '\n') break
      }
      blank(ends)
      if (source[i] === one) {
        out += one
        i += 1
      }
      continue
    }

    if (one === '`') {
      out += one
      i += 1
      while (i < source.length) {
        if (source[i] === '\\') {
          blank(i + 2)
          continue
        }
        if (source[i] === '`') {
          out += '`'
          i += 1
          break
        }
        if (source.slice(i, i + 2) === '${') {
          out += '${'
          i += 2
          const inside = scan(source, i, true)
          out += inside.out
          i = inside.next
          if (source[i] === '}') {
            out += '}'
            i += 1
          }
          continue
        }
        blank(i + 1)
      }
      continue
    }

    out += one
    i += 1
  }
  return { out, next: i }
}

/** Where the block opening at `from` ends, one past the brace that closes it. */
function braced(text, from) {
  let depth = 0
  for (let i = from; i < text.length; i += 1) {
    if (text[i] === '{') depth += 1
    else if (text[i] === '}') {
      depth -= 1
      if (!depth) return i + 1
    }
  }
  return text.length
}

const CATCH = /(^|[^.\w$])catch\s*(\(\s*([A-Za-z0-9_$]+)[^)]*\)\s*)?\{/g

/**
 * Every `catch` clause of one source: the name it binds, its body as written,
 * and the body's code alone.
 *
 * `.catch(…)` is a different construct and is left alone. It takes a function,
 * and the same question is asked of that function's body wherever it stands.
 */
export function clauses(source) {
  const over = masked(source)
  const found = []
  for (const one of over.matchAll(CATCH)) {
    const opens = one.index + one[0].length - 1
    const ends = braced(over, opens)
    found.push({
      binding: one[3] ?? '',
      body: source.slice(opens + 1, ends - 1),
      code: over.slice(opens + 1, ends - 1),
      at: source.slice(0, one.index).split('\n').length,
    })
  }
  return found
}

/** Whether the body does anything at all with the error it was handed. */
const reads = (one) => one.binding !== '' && new RegExp(`\\b${one.binding}\\b`).test(one.code)

/**
 * The lines the catch clauses of one source open on that throw the error away
 * and say nothing about why.
 */
export function silent(source) {
  return clauses(source)
    .filter((one) => !reads(one) && !/\/\/|\/\*/.test(one.body))
    .map((one) => one.at)
}
