/**
 * A template ref that is asked for is a template ref that is bound.
 *
 * `useTemplateRef('page')` answers whatever the template put `ref="page"` on,
 * and nothing where no template put it anywhere. Nothing fails: the ref is
 * `null`, every reader is written `page.value?.…`, and the window quietly does
 * less than it says. Only a person opening the application finds out.
 *
 * A component's own template is the one that binds it, and a composable's is
 * the template of whichever component calls the composable — so the name is
 * looked for across the package rather than in one file.
 */
import { blocks, code, sources } from './source.mjs'

// The type is written between angle brackets and holds angle brackets of its
// own, so it is read up to the bracket that opens the call.
const ASKED = /useTemplateRef\s*(?:<[^(]*>)?\s*\(\s*'([^']+)'/g
// A `:ref` or `v-bind:ref` is bound to whatever the window decides while it
// runs, and has no written name to read.
const BOUND = /(^|[\s>])ref\s*=\s*"([^"]+)"/g

/**
 * Every template ref name one file asks for.
 *
 * The name is in a string, and blanking the comments blanks the strings with
 * them. What is read is where the blanked text still says `useTemplateRef` —
 * a commented one says nothing there — and the name is taken from the same
 * place in the text as written, which the blanking leaves standing.
 */
export function asks(text) {
  const script = text.includes('<script') ? blocks(text, 'script').join('\n') : text
  const read = code(script)
  const found = []
  for (const one of read.matchAll(/useTemplateRef/g)) {
    ASKED.lastIndex = one.index
    const said = ASKED.exec(script)
    if (said?.index === one.index) found.push(said[1])
  }
  return found
}

/**
 * Every template ref name one file binds.
 *
 * The markup is what is left when the script and the style are taken out. A
 * component's template holds templates of its own where it fills a slot, so
 * reading to the first closing tag reads only as far as the first slot.
 */
export function binds(text) {
  let markup = text
  for (const tag of ['script', 'style']) {
    for (const one of blocks(text, tag)) markup = markup.replace(one, '')
  }
  return [...markup.matchAll(BOUND)].map((one) => one[2])
}

/** Which package a file stands in, which is as far as a template ref reaches. */
export function packageOf(at) {
  const parts = at.split('/')
  return parts.slice(0, parts.indexOf('src')).join('/')
}
