/**
 * A function is named by an imperative verb phrase — not a third-person verb,
 * not a gerund, not a participle.
 *
 * `get`, `read`, `render`, `resolve` say what the caller is asking for.
 * `carries`, `holds`, `shows` read as a narrator describing somebody else's
 * code, and `dressing`, `minting`, `offered` name the doing of a thing rather
 * than the doing of it. A predicate is the other shape a function takes, and
 * reads `is`, `has` or `can`.
 *
 * A preposition standing alone is refused where the name travels: `At`, `Under`
 * and `Beside` say where a thing is, and a caller in another package has to
 * read the declaration before the name means anything. A name read beside its
 * one use stands in the sentence that explains it, and the type rule in
 * `container/nouns_test.go` draws the same line.
 *
 * A machine reading a name can only see how it ends, and no suffix tells a
 * third-person verb from a plural noun, or a gerund from a word that merely
 * finishes those letters. So the words that are not verb forms are written
 * down, with what each one means. It is a dictionary, not a list of
 * exemptions: a reader can apply the rule without reading it, and a word that
 * has left the source has to leave the dictionary.
 */
import { code } from './source.mjs'

/**
 * baseline are the Go functions still named by a gerund or a participle, and
 * the list only shrinks. A name here is debt somebody wrote down; a name that
 * has left the source has to leave this list, and the test below refuses one
 * that nothing is called any more.
 */
export const goBaseline = []

/**
 * The words of ours ending in -ing or -ed that name a thing rather than the
 * doing of one, and what each means. A factory here is free to take a plain
 * noun, and several of the windows' do.
 *
 * A base-form verb that merely finishes in those letters stands here too: no
 * machine can tell `embed` from `offered` by its ending.
 */
export const nouns = {
  bring: 'to carry something to where the caller is',
  ceiling: 'the number a count is not let past',
  drawing: 'a picture',
  embed: 'to turn a text into the direction that stands for it',
  feed: 'to hand something on to what is waiting for it',
  heading: 'a line a section of a note stands under',
  landing: 'where something let go comes to rest',
  opening: 'the way in, and how wide it is',
  reading: 'what a document was read as',
  recording: 'a sound file',
  routing: 'the way a line is taken from one place to another',
  seed: 'the number a run of chance is started from',
  sibling: 'a note under the same parent',
  spacing: 'the room left between things',
  timing: 'when something happens, and how long it takes',
}

/**
 * baseline are the functions still named by a gerund or a participle, and the
 * list only shrinks. A name here is debt somebody wrote down; a name that has
 * left the source has to leave this list, and the test below refuses one that
 * nothing is called any more.
 */
export const baseline = []

/** The words of one name, as a reader says them. */
export const words = (name) =>
  [...name.matchAll(/[A-Z]+(?![a-z])|[A-Z][a-z0-9]*|^[a-z][a-z0-9]*/g)].map((one) =>
    one[0].toLowerCase(),
  )

/**
 * Whether a word reads as a verb form a function may not be named by.
 *
 * Two letters is the shortest an English verb runs to — owing, doing, being —
 * so one letter in front of the ending is a word that ends there by accident:
 * a ring is a thing. The past forms that carry no ending are listed below.
 *
 * A third-person verb is not tested for. `carries` and `cells` end the same
 * way, and a factory here is free to take a plain noun, so no machine can tell
 * the narrator from the thing. A person reads those.
 */
export const verbal = (word) => /^.{2,}(ing|ed)$/.test(word) || past.has(word)

/**
 * The past forms no ending gives away. `drawn`, `held` and `took` narrate as
 * plainly as `offered` does, and a machine reading suffixes cannot see it, so
 * they are written down. English has a fixed number of these, and this is them.
 *
 * A form that doubles as the base is not here — `read`, `run`, `set`, `put`,
 * `cut`, `hit`, `split`, `shut`, `spread`, `cost`, `let` — because
 * `readTable` is a caller asking and nothing in the name says otherwise. Nor
 * is `left`, which is a side. A word that lands here and means something else
 * goes in the dictionary above, with what it means.
 */
const past = new Set([
  'began', 'begun', 'blown', 'borne', 'bought', 'broke', 'broken', 'brought',
  'built', 'burnt', 'came', 'caught', 'chose', 'chosen', 'dealt', 'done',
  'drawn', 'drew', 'driven', 'drove', 'eaten', 'fallen', 'fell', 'felt',
  'flew', 'flown', 'forgave', 'forgiven', 'forgot', 'forgotten', 'froze',
  'frozen', 'gave', 'given', 'gone', 'got', 'gotten', 'grew', 'grown',
  'heard', 'held', 'hid', 'hidden', 'kept', 'knew', 'known', 'laid', 'lain',
  'lent', 'lost', 'made', 'meant', 'met', 'paid', 'ran', 'rang', 'risen',
  'rung', 'said', 'sang', 'sank', 'sat', 'seen', 'sent', 'shaken', 'shone',
  'shook', 'shown', 'shrunk', 'slept', 'slid', 'sold', 'sought', 'sown',
  'spent', 'spoke', 'spoken', 'spun', 'stole', 'stolen', 'stood', 'struck',
  'stuck', 'swept', 'swam', 'sworn', 'swum', 'taken', 'taught', 'thought',
  'threw', 'thrown', 'told', 'took', 'tore', 'torn', 'understood', 'went',
  'woke', 'woken', 'won', 'wore', 'worn', 'wove', 'woven', 'written', 'wrote',
])

/**
 * The prepositions. A name that is one of these and nothing else says where a
 * thing stands: `At(list, 2)`, `Under(path)` and `Beside(a, b)` each need the
 * declaration read before the name means anything.
 *
 * A preposition with a word behind it is a phrase and is not read here.
 * `InOrder`, `withColumn` and `onKeyDown` are names; telling those from
 * `atFoot` needs a person.
 *
 * `round` is not here. It is the verb a number is rounded by, and every name
 * this repository gives it is that one.
 */
const prepositions = new Set([
  'about', 'above', 'across', 'after', 'against', 'ahead', 'along', 'alongside',
  'among', 'apart', 'around', 'aside', 'at', 'away', 'before', 'behind',
  'below', 'beneath', 'beside', 'besides', 'between', 'beyond', 'by', 'down',
  'during', 'except', 'for', 'from', 'in', 'inside', 'into', 'near', 'off',
  'on', 'onto', 'out', 'outside', 'over', 'past', 'since', 'through',
  'throughout', 'till', 'to', 'toward', 'towards', 'under', 'underneath',
  'until', 'up', 'upon', 'via', 'with', 'within', 'without',
])

/**
 * Whether a name is refused. The first word is what is read: it is the verb,
 * and the rest says what the verb acts on. `getSettings` asks for settings and
 * `onDropEntries` answers a drop; both end in a plural noun and neither is a
 * narrator. `carries` and `dressing` have nothing in front of the verb form,
 * so the verb form is the name.
 *
 * A verb form later in a name is not read here. `getRenamedPath` is a
 * participle used as an adjective and is right; `faceAdded` is a narrator and
 * is wrong, and telling them apart needs a list of every English verb. The
 * `naming-reviewer` role is what stands there.
 *
 * travels says whether the name is read anywhere but beside its declaration,
 * and only such a name is held to the preposition rule.
 */
export function refused(name, travels = true) {
  const said = words(name)
  const first = said[0] ?? ''
  if (travels && said.length === 1 && prepositions.has(first)) return true
  if (!verbal(first)) return false
  return !Object.hasOwn(nouns, first)
}

/**
 * Whether a Go name travels. Go says it in the name: a declaration beginning
 * with a capital is served to every package that imports this one.
 */
export const goTravels = (name) => /^[A-Z]/.test(name)

/**
 * Every function one file declares: a `function`, and a `const` or `let` whose
 * value is an arrow or a function. A method is outside this — it is read as
 * part of the object it stands in.
 *
 * A parameter list is followed one bracket deep, which is as far as a default
 * value goes: `(deps = support())` is a signature and `(a) => (b) =>` is two.
 *
 * A return type is read up to the arrow and not to the first `>`, so a generic
 * one — `Promise<DeckPresetResult>` — is a return type and not the end of the
 * declaration.
 */
const DECLARED =
  /(?:^|\n)[ \t]*(?<served>export\s+)?(?:async\s+)?function\s+(?<named>[a-z][\w$]*)(?:<[^>(]*>)?\s*(?<list>\((?:[^()]|\([^()]*\))*\))|(?:^|\n)[ \t]*(?<sent>export\s+)?(?:const|let)\s+(?<held>[a-z][\w$]*)(?::[^=\n]*)?\s*=\s*(?:async\s+)?(?<takes>\((?:[^()]|\([^()]*\))*\)|[a-z][\w$]*)\s*(?::(?:[^=\n]|=(?!>))*)?=>/g

export function declares(source) {
  const found = []
  for (const one of code(source).matchAll(DECLARED)) {
    found.push(one.groups.named ?? one.groups.held)
  }
  return found
}

/**
 * The functions of one file that travel. A module says it in a word: a
 * declaration carrying `export` is read by whoever imports the file, and one
 * without it is read nowhere but here.
 */
export function declaresAbroad(source) {
  const found = []
  for (const one of code(source).matchAll(DECLARED)) {
    if (one.groups.served ?? one.groups.sent) found.push(one.groups.named ?? one.groups.held)
  }
  return found
}

/**
 * What a parameter is called, read off the front of one entry of a list: the
 * name, and then a type, a default, or the end of it.
 */
const PARAMETER = /^\s*(?:readonly\s+)?(?<takes>[a-z][\w$]*)\s*\??\s*(?::|=(?!>)|$)/

/**
 * One list of parameters cut at its own commas. A comma inside a type or a
 * destructuring belongs to that, so the depth is counted — and `=>` is an
 * arrow and not a bracket closing.
 */
function apart(list) {
  const out = []
  let depth = 0
  let from = 0
  for (let at = 0; at < list.length; at += 1) {
    const one = list[at]
    if ('([{<'.includes(one)) depth += 1
    else if (')]}'.includes(one)) depth -= 1
    else if (one === '>' && list[at - 1] !== '=') depth -= 1
    else if (one === ',' && depth === 0) {
      out.push(list.slice(from, at))
      from = at + 1
    }
  }
  out.push(list.slice(from))
  return out
}

/**
 * Every parameter the declared functions of one file take. A caller reads a
 * parameter the way it reads the function, so the same rule holds for both. A
 * destructured list is skipped: those are the members of a contract declared
 * elsewhere, and they are read there.
 */
export function takes(source) {
  const found = []
  for (const one of code(source).matchAll(DECLARED)) {
    const list = one.groups.list ?? one.groups.takes
    if (!list?.startsWith('(')) continue
    for (const part of apart(list.slice(1, -1))) {
      const name = PARAMETER.exec(part)?.groups.takes
      if (name) found.push(name)
    }
  }
  return found
}

const GO_DECLARED = /^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)/gm

/**
 * The method names Go's own interfaces fix. A type satisfies `context.Context`
 * by declaring `Done`, and a name the standard library chose is not a name this
 * repository is free to write differently.
 */
const given = new Set([
  'Close', 'Done', 'Err', 'Error', 'Kind', 'Len', 'Less', 'Next', 'Read',
  'Reset', 'Scan', 'Seek', 'String', 'Swap', 'Type', 'Value', 'Write',
])

/**
 * Every function and method one Go file declares, less the ones Go names for
 * it. gofmt puts a declaration of the file's own at the left margin and nothing
 * else, so a margin is all this has to read.
 *
 * A method is read here where a TypeScript one is not: Go hangs it on a type by
 * a receiver and it is declared at the margin like any other function, so the
 * caller reads `day.GetName()` exactly as it reads `getName()`.
 */
export function goDeclares(source) {
  return [...code(source, true).matchAll(GO_DECLARED)]
    .map((one) => one[1])
    .filter((name) => !given.has(name))
}
