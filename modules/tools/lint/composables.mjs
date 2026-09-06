/**
 * A function named `useX` deals in reactive state.
 *
 * `use` is Vue's word for a composable, and a composable is a function that
 * takes reactive state, returns it, or holds it across renders. The prefix
 * says a caller is getting one. It does not say a lifecycle hook is
 * registered, and a composable registering none is a composable still.
 *
 * The reverse is not a rule, so this walk runs one way. A factory that hands
 * back refs and can be called anywhere is free to take a plain noun, and
 * several of the windows' do. What is refused is the direction that misleads:
 * `use` on a function dealing in no reactive state, which promises a
 * composable and hands back a plain value.
 *
 * Read off the declaration's own code, because that is what a reader has. A
 * composable either names Vue's reactivity, or names a ref in its signature,
 * or reads a `.value`, or holds another composable — there is no fifth way to
 * deal in reactive state, and a declaration doing none of them deals in none.
 *
 * A `useX` is a function or the value of a name. A method is outside this: the
 * tree declares no composable as one.
 */
import { code } from './source.mjs'

/** How a declaration deals in reactive state, in the only ways there are. */
const REACTIVE = new RegExp(
  [
    // State made, watched, or scoped.
    String.raw`\b(?:ref|shallowRef|reactive|shallowReactive|computed)\s*[(<]`,
    String.raw`\b(?:customRef|toRef|toRefs|toValue|effectScope)\s*[(<]`,
    String.raw`\bwatch(?:Effect|PostEffect|SyncEffect)?\s*\(`,
    String.raw`\bon(?:ScopeDispose|Mounted|BeforeMount|Unmounted|BeforeUnmount)\s*\(`,
    String.raw`\bon(?:Updated|BeforeUpdate|Activated|Deactivated|ErrorCaptured)\s*\(`,
    String.raw`\bon(?:ServerPrefetch|RenderTracked|RenderTriggered)\s*\(`,
    String.raw`\b(?:provide|inject|getCurrentInstance|getCurrentScope)\s*\(`,
    // State named in the signature.
    String.raw`\b(?:Shallow|Computed|WritableComputed)?Ref\s*<`,
    String.raw`\bMaybeRef(?:OrGetter)?\b`,
    // State read or written.
    String.raw`\.value\b`,
    // Another composable held.
    String.raw`\buse[A-Z]\w*\s*\(`,
  ].join('|'),
)

/** Where a `useX` is declared, as a function or as the value of a name. */
const DECLARED = /(?:^|\n)[ \t]*(?:export\s+)?(?:async\s+)?(?:function|const|let)\s+(use[A-Z]\w*)/g

/** The indentation of the line `at` stands on. */
const indented = (text, at) => {
  const line = text.lastIndexOf('\n', at - 1) + 1
  return /^[ \t]*/.exec(text.slice(line, at))[0].length
}

/**
 * A declaration from `from` onwards: its signature and its body together,
 * ending where the block it opened closes. A brace-less arrow opens no block,
 * and its body is the expression, which ends at the next line no deeper than
 * the declaration.
 *
 * A signature written over several lines closes its parameters at the margin,
 * so where a declaration ends cannot be read off the indentation alone:
 * `useDrag` ends its parameters on a line of its own, and a walk stopping at
 * the first close in column one reads the signature and none of the body.
 */
export function declaration(text, from) {
  const own = indented(text, from)
  let depth = 0
  let block = false
  for (let i = from; i < text.length; i++) {
    const letter = text[i]
    if (letter === '{' && depth === 0 && !block) block = true
    if (letter === '(' || letter === '[' || letter === '{') depth += 1
    else if (letter === ')' || letter === ']' || letter === '}') {
      depth -= 1
      if (block && depth <= 0) return text.slice(from, i + 1)
    } else if (depth === 0 && (letter === ';' || letter === '\n')) {
      const rest = text.slice(i + 1)
      const next = /^[ \t]*/.exec(rest)[0].length
      const blank = /^[ \t]*(\n|$)/.test(rest)
      if (letter === ';' || (!blank && next <= own)) return text.slice(from, i)
    }
  }
  return text.slice(from)
}

/** Whether a declaration deals in reactive state. */
const reactive = (text) => REACTIVE.test(text)

/**
 * Every `useX` one file declares, and whether each deals in reactive state.
 * Comments and strings are blanked first: a `.value` a string says is no read.
 *
 * Read from past its own name: a composable held is one of the four ways of
 * dealing in reactive state, and a declaration left to read its own head
 * answers that it holds itself.
 */
export function composables(text) {
  const found = []
  const read = code(text)
  for (const one of read.matchAll(DECLARED)) {
    const said = declaration(read, one.index + one[0].length)
    found.push({ name: one[1], reactive: reactive(said) })
  }
  return found
}
