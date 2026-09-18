/**
 * A field holding a boolean says so in its name: `isStale`, `hasTranscript`,
 * `canRead`. A bare adjective or participle reads as a value of some other
 * kind — `changed` is a past tense, `disabled` is a state somebody set, and
 * neither says that what stands there is true or false.
 *
 * The rule holds in both languages. Go's own idiom agrees for a predicate —
 * `IsDir`, `IsAbs` — and says nothing about a field, so this is the
 * repository's rule and not the language's.
 */
import { code } from './source.mjs'

/**
 * How a boolean name may begin. `is`, `has` and `can` are the three §6 names.
 * The rest are the other shapes a name takes and still says true or false:
 * `shouldRebuild` is asked for, `wasWritten` happened, `showEdgeLabels` draws.
 */
const PREFIX = /^(is|has|can|should|must|will|does|allows|was|were|show)([A-Z0-9]|$)/i

/**
 * The names that mirror something this repository did not name: a field of a
 * browser event matched against what the browser hands over, and a component
 * prop bound straight to the attribute of the same name on the element it
 * draws. `<Button disabled>` is read as the HTML it becomes, and the styling
 * that answers `disabled:` reads the attribute rather than the prop.
 *
 * Nothing else belongs here. A field of ours carrying a schema's word is still
 * ours to name, because the mapping between the two is written by hand.
 */
const borrowed = new Set([
  'altKey', 'autofocus', 'checked', 'ctrlKey', 'disabled', 'expanded',
  'focused', 'focusVisible', 'hidden', 'metaKey', 'multiple', 'open',
  'readonly', 'required', 'selected', 'shiftKey',
])

/** A TypeScript field or property declared boolean, by the name it is given. */
const SAID = /(?:^|\n)[\t ]*(?:readonly\s+)?([a-z][\w$]*)\??\s*:\s*(?:boolean|Readonly<Ref<boolean>>|Ref<boolean>)\s*(?=[,;\n)])/g

/**
 * A Go struct field declared bool, by the name it is given. Only the body of a
 * `struct {` is read: a signature broken over several lines indents its
 * parameters the same way, and a parameter is not a field.
 */
const GO_SAID = /(?:^|\n)[\t ]*([A-Za-z][\w]*)((?:,\s*[A-Za-z][\w]*)*)\s+bool\s*(?:`[^`]*`)?\s*(?:\/\/.*)?$/gm

/** Whether a name says that what stands under it is true or false. */
export const saysBoolean = (name) => PREFIX.test(name) || borrowed.has(name)

/** Every boolean field one TypeScript or Vue file declares. */
export function declaresBooleans(source) {
  return [...code(source).matchAll(SAID)].map((one) => one[1])
}

/** The body of every `struct {` one file declares, braces counted. */
function structBodies(source) {
  const found = []
  const opens = /\bstruct\s*\{/g
  for (const one of source.matchAll(opens)) {
    let depth = 1
    let i = one.index + one[0].length
    const from = i
    while (i < source.length && depth > 0) {
      if (source[i] === '{') depth += 1
      else if (source[i] === '}') depth -= 1
      i += 1
    }
    found.push(source.slice(from, i - 1))
  }
  return found
}

/** Every boolean field one Go file declares, each struct body read on its own. */
export function goDeclaresBooleans(source) {
  const found = []
  for (const body of structBodies(code(source))) {
    for (const one of body.matchAll(GO_SAID)) {
      found.push(one[1], ...(one[2] ? one[2].split(',').map((two) => two.trim()).filter(Boolean) : []))
    }
  }
  return found
}

/**
 * baseline are the boolean fields still named without one of the words, each
 * under the file it stands in, and the list only shrinks.
 *
 * Go carries none, and each of the four left is a name this repository does not
 * own. `VaultCounts.reading` and the card's `seen` are the shape of a generated
 * message, matched against the client by their fields; `evenLoad` is the word
 * the settings file writes. Renaming one of these is a change to what crosses
 * the wire or what is read from disk, which is a decision and not a rename.
 */
export const baseline = [
  'modules/apps/desktop/editor/src/entities/deck/lib/presets.ts declares evenLoad',
]
