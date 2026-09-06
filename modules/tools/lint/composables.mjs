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
 * Read off the declaration's own text, because that is what a reader has. A
 * composable either names Vue's reactivity, or names a ref in its signature,
 * or reads a `.value`, or holds another composable — there is no fifth way to
 * deal in reactive state, and a declaration doing none of them deals in none.
 */

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

/**
 * A declaration from `from` onwards: its signature and its body together,
 * ending where the body it opened closes.
 *
 * A signature written over several lines closes its parameters at the margin,
 * so where a declaration ends cannot be read off the indentation: `useDrag`
 * ends its parameters on a line of its own, and a walk stopping at the first
 * close in column one reads the signature and none of the body.
 */
export function declaration(text, from) {
  let braces = 0
  let opened = false
  for (let i = from; i < text.length; i++) {
    const letter = text[i]
    if (letter === '{') {
      braces += 1
      opened = true
    } else if (letter === '}') {
      braces -= 1
      if (opened && braces <= 0) return text.slice(from, i + 1)
    }
  }
  return text.slice(from)
}

/** Whether a declaration deals in reactive state. */
const reactive = (text) => REACTIVE.test(text)

/**
 * Every `useX` one file declares, and whether each deals in reactive state.
 *
 * Read from past its own name: a composable held is one of the four ways of
 * dealing in reactive state, and a declaration left to read its own head
 * answers that it holds itself.
 */
export function composables(text) {
  const found = []
  for (const one of text.matchAll(DECLARED)) {
    const said = declaration(text, one.index + one[0].length)
    found.push({ name: one[1], reactive: reactive(said) })
  }
  return found
}
