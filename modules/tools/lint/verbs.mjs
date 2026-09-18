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
 * A third-person verb carries no ending a machine can read: `carries` and
 * `cells` end the same way. The words are written down in `narrators` instead.
 */
export const verbal = (word) => /^.{2,}(ing|ed)$/.test(word) || past.has(word)

/**
 * The third-person verbs. `holds` and `recognises` read as a narrator
 * describing somebody else's code, where `hold` and `recognise` are the caller
 * asking for something.
 *
 * A denylist, where the dictionary above is an allowlist, because that is the
 * shorter list by three times: the words ending in -s that are plural nouns run
 * past a hundred and fifty and grow with every new one, and these do not.
 *
 * A word here is refused whichever way it is read. `covers` and `shares` are
 * plural nouns as well as verbs, and a function named by a plain noun is
 * refused too — `getCovers` says what the caller gets, and `covers` says it of
 * neither reading.
 *
 * `contains` and `matches` are not here. The standard library named both, and a
 * caller reads them as that library's word rather than as a narrator's.
 */
const narrators = new Set([
  'admits', 'allows', 'closes', 'covers', 'crosses', 'deals', 'declares',
  'divides', 'encloses', 'follows', 'forgets', 'hangs', 'holds', 'joins',
  'keeps', 'knows', 'lands', 'leads', 'learns', 'lives', 'merges', 'opens',
  'presses', 'proofreads', 'reaches', 'reckons', 'recognises', 'refuses',
  'repeats', 'ripens', 'runs', 'says', 'shares', 'sits', 'spends', 'spreads',
  'stands', 'stops', 'supports', 'takes', 'tells', 'transcribes', 'writes',
])

/**
 * Whether a name reads as a narrator. Only a function is held to this: a
 * parameter is routinely a plural noun — `marksIn(runs, origin)` hands over
 * runs — and the name carries no verb in front to tell the two apart.
 *
 * The first word is what is read, as everywhere else: `keepsNotes` narrates and
 * `keepNotes` asks.
 */
export const narrates = (name) => narrators.has(words(name)[0] ?? '')

/**
 * baseline are the functions still named by a third-person verb, in either
 * language, and the list only shrinks. A name here is debt somebody wrote down;
 * a name that has left the source has to leave this list, and the test below
 * refuses one that nothing is called any more.
 */
export const narratorBaseline = []

/**
 * The words ending in -ly or -est that are not an adverb or a superlative.
 * `apply` and `request` are verbs a caller says, `family` and `tally` are
 * things, and `honest` is what a thing is rather than which of several it is.
 *
 * Six words, where the endings they are exceptions to catch twenty: this is the
 * allowlist worth writing, and the plural nouns ending in -s are not.
 */
const notDegrees = new Set(['apply', 'family', 'honest', 'reply', 'request', 'tally'])

/**
 * The words ending in -able that a caller says rather than a thing is.
 * `enable` and `disable` are what is done to a setting; the rest of the ending
 * marks an adjective every time it appears here.
 */
const notAdjectives = new Set(['disable', 'enable', 'unable'])

/**
 * Whether a name says what something is like, how, or how much, rather than
 * what is done. `plainly`, `quietest` and `readable` describe; `getPlainRune`,
 * `findQuietest` and `canRead` are asked for.
 *
 * Only a function is held to this. A parameter naming a bound is an adjective
 * by nature — `cut(scores, longest, shortest)` hands over the longest and the
 * shortest a run may be — and there is no verb in front to tell it apart.
 */
export const describes = (name) => {
  const first = words(name)[0] ?? ''
  if (/^.{3,}[ai]ble$/.test(first)) return !notAdjectives.has(first)
  return /^.{3,}(ly|est)$/.test(first) && !notDegrees.has(first)
}

/**
 * The words that stand for a thing instead of naming it: pronouns, and the
 * determiners and adverbs of place and degree. `one`, `all`, `ours` and `here`
 * each need the declaration read before the name says anything, which is the
 * same fault a preposition alone has.
 *
 * The list is closed. English gains no new pronouns, so nothing is ever added
 * here for a word somebody wrote.
 */
const standIns = new Set([
  'again', 'all', 'alone', 'another', 'anything', 'anywhere', 'both', 'each',
  'either', 'enough', 'every', 'everything', 'everywhere', 'far', 'here',
  'itself', 'least', 'less', 'mine', 'more', 'most', 'neither', 'none',
  'nothing', 'nowhere', 'one', 'only', 'other', 'others', 'ours', 'quite',
  'rather', 'same', 'somewhere', 'still', 'such', 'that', 'theirs', 'there',
  'these', 'this', 'those', 'too', 'very', 'yours',
])

/**
 * Whether a name stands for a thing rather than naming it. As with the
 * preposition rule, only a whole name is read: `one` says nothing and
 * `getOnlyVault` says what comes back.
 */
export const standsFor = (name) => {
  const said = words(name)
  return said.length === 1 && standIns.has(said[0])
}

/**
 * baseline are the functions still named by an adverb or a superlative, and the
 * list only shrinks. A name here is debt somebody wrote down.
 */
export const degreeBaseline = []

/**
 * baseline are the functions still named by a word that stands for a thing
 * rather than naming it, and the list only shrinks.
 */
export const standInBaseline = []

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
  'during', 'except', 'for', 'from', 'in', 'inside', 'into', 'near', 'of',
  'off', 'on', 'onto', 'out', 'outside', 'over', 'past', 'since', 'than',
  'through', 'throughout', 'till', 'to', 'toward', 'towards', 'under',
  'underneath', 'until', 'up', 'upon', 'via', 'with', 'within', 'without',
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
  return [...code(source).matchAll(GO_DECLARED)]
    .map((one) => one[1])
    .filter((name) => !given.has(name))
}
