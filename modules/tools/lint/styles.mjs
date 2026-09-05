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

/**
 * The class selectors a style block declares.
 *
 * A selector inside `:deep()` is left out: it addresses the DOM of a component
 * this one mounts, which this template does not write the classes of.
 */
export function declares(style) {
  const css = style
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/'[^'\n]*'|"[^"\n]*"/g, "''")
    .replace(/@import[^;]*;/g, '')
    .replace(/:deep\([^()]*\)/g, '')
  const found = new Set()
  for (const one of css.matchAll(/\.(-?[_a-zA-Z][\w-]*)/g)) found.add(one[1])
  return found
}

/**
 * The classes everything but the style block can put on an element: what is
 * written out in a string, and the stem of one built by putting a value after
 * a prefix.
 */
export function sets(rest) {
  const named = new Set()
  const quoted = /'([^'\n]*)'|"([^"\n]*)"|`([^`]*)`/g
  for (const one of rest.matchAll(quoted)) words(one[1] ?? one[2] ?? one[3] ?? '', named)

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
