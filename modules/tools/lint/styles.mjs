/**
 * A class a component styles and no template of its can set.
 *
 * A custom property declared on a component's root class reaches its children
 * through that class and nowhere else, so a root renamed in the template and
 * left in the style block goes on parsing, goes on passing every story that
 * asserts an arrangement, and draws at the wrong size in the window.
 */

/** The classes Vue writes itself for a transition it was given a name. */
const STAGES = [
  'enter-from',
  'enter-active',
  'enter-to',
  'leave-from',
  'leave-active',
  'leave-to',
  'move',
]

const words = (text, into) => {
  for (const one of text.matchAll(/[A-Za-z_-][\w-]*/g)) into.add(one[0])
  return into
}

/** What each of `text`'s bracketed arguments comes to, brackets and all. */
const bracketed = (text, at) => {
  let depth = 0
  for (let i = at; i < text.length; i += 1) {
    if (text[i] === '(') depth += 1
    else if (text[i] === ')') {
      depth -= 1
      if (!depth) return i + 1
    }
  }
  return text.length
}

/** The reaching selectors, in every spelling, with what they address. */
const REACHING = /::?v-(?:deep|global|slotted)\b|:(?:deep|global|slotted)\b/

/**
 * `css` with every reaching selector taken out, argument and all.
 *
 * A nested bracket is followed rather than stopped at, and the bare-combinator
 * spelling — `.a ::v-deep .x` — takes the rest of its selector with it.
 */
const reaching = (css) => {
  let out = ''
  let rest = css
  for (let found = rest.match(REACHING); found; found = rest.match(REACHING)) {
    out += rest.slice(0, found.index)
    const after = found.index + found[0].length
    if (rest[after] === '(') {
      rest = rest.slice(bracketed(rest, after))
      continue
    }
    // Nothing bracketed: what follows is addressed through it, to the end of
    // this selector.
    const ends = rest.slice(after).search(/[,{]/)
    rest = ends < 0 ? '' : rest.slice(after + ends)
  }
  return out + rest
}

/**
 * The class selectors a style block declares.
 *
 * A selector inside `:deep()`, `:global()` or `:slotted()` is left out: it
 * addresses the DOM of a component this one mounts or is mounted in, which this
 * template does not write the classes of.
 */
export function declares(style) {
  const css = reaching(
    style
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/'[^'\n]*'|"[^"\n]*"/g, "''")
      .replace(/@import[^;]*;/g, ''),
  )
  const found = new Set()
  for (const one of css.matchAll(/\.(-?[_a-zA-Z][\w-]*)/g)) found.add(one[1])
  return found
}

/**
 * The classes everything but the style block can put on an element: what is
 * written out in a string, and the stem of one built by putting a value after
 * a prefix.
 *
 * A module specifier is not a string the template can reach: a component in
 * `welcome/` importing from `'./welcome'` would otherwise look able to set
 * `.welcome`, which is most components and the fault this file exists for.
 *
 * The kebab-case of a component the template mounts is a class it can set. A
 * child's root element carries the scope of the parent that mounted it, so
 * `.card__value > .divider` reaches the root of a `<Divider>` this template
 * writes.
 */
export function sets(source) {
  const rest = source.replace(/\b(?:from|import)\s*\(?\s*(['"])[^'"\n]*\1/g, '')
  const named = new Set()
  const quoted = /'([^'\n]*)'|"([^"\n]*)"|`([^`]*)`/g
  for (const one of rest.matchAll(quoted)) words(one[1] ?? one[2] ?? one[3] ?? '', named)

  for (const one of rest.matchAll(/<([A-Z][A-Za-z0-9]*)\b/g)) {
    named.add(one[1].replace(/(?<=.)(?=[A-Z])/g, '-').toLowerCase())
  }

  for (const one of rest.matchAll(/<[Tt]ransition(?:-?[Gg]roup)?\b[^>]*\bname="([^"]+)"/g)) {
    for (const stage of STAGES) named.add(`${one[1]}-${stage}`)
  }

  const stems = []
  for (const one of rest.matchAll(/`([^`$]*)\$\{/g)) if (one[1]) stems.push(one[1])
  for (const one of rest.matchAll(/'([^'\n]*)'\s*\+/g)) if (one[1]) stems.push(one[1])
  return { named, stems }
}

/** The classes one component styles and cannot reach, in the order declared. */
export function unreachable(source) {
  const style = []
  const opens = /<style(\s[^>]*)?>([\s\S]*?)<\/style>/g
  for (const one of source.matchAll(opens)) style.push(one[2])
  if (!style.join('').trim()) return []

  const { named, stems } = sets(source.replace(opens, ''))
  return [...declares(style.join('\n'))].filter(
    (one) => !named.has(one) && !stems.some((stem) => one.startsWith(stem)),
  )
}
