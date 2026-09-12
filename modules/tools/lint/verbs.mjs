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
 * A machine reading a name can only see how it ends, and no suffix tells a
 * third-person verb from a plural noun, or a gerund from a word that merely
 * finishes those letters. So the words that are not verb forms are written
 * down, with what each one means. It is a dictionary, not a list of
 * exemptions: a reader can apply the rule without reading it, and a word that
 * has left the source has to leave the dictionary.
 */
import { code } from './source.mjs'

/**
 * The words of ours ending in -ing or -ed that name a thing rather than the
 * doing of one, and what each means. A factory here is free to take a plain
 * noun, and several of the windows' do.
 */
export const nouns = {
  drawing: 'a picture',
  heading: 'a line a section of a note stands under',
  landing: 'where something let go comes to rest',
  opening: 'the way in, and how wide it is',
  reading: 'what a document was read as',
  recording: 'a sound file',
  routing: 'the way a line is taken from one place to another',
  spacing: 'the room left between things',
  timing: 'when something happens, and how long it takes',
}

/**
 * baseline are the functions still named by a gerund or a participle, and the
 * list only shrinks. A name here is debt somebody wrote down; a name that has
 * left the source has to leave this list, and the test below refuses one that
 * nothing is called any more.
 */
export const baseline = [
  'adding', 'aiming', 'allowed', 'answered', 'answeredOn', 'answering', 'applied', 'arranged',
  'arrived', 'arriving', 'asked', 'asking', 'attached', 'blocking', 'bounded', 'braced',
  'bring', 'carried', 'changed', 'choosing', 'chorded', 'clamped', 'cleaned', 'clocked',
  'coded', 'counted', 'crowded', 'crowdingFor', 'cutting', 'dated', 'declared', 'divided',
  'doing', 'dragged', 'draggedIn', 'dropped', 'edited', 'emptied', 'ended', 'escaped',
  'failed', 'feeding', 'fetching', 'filled', 'filledPercent', 'filling', 'finished', 'folded',
  'followed', 'following', 'framed', 'framedIn', 'grouped', 'handed', 'handedOn', 'handling',
  'holding', 'hovered', 'hurried', 'joined', 'keeping', 'keyed', 'knobbed', 'learned',
  'listed', 'listing', 'making', 'marked', 'marking', 'matching', 'measured', 'measuring',
  'measuringContext', 'merged', 'moved', 'named', 'naming', 'nested', 'nothing', 'noticed',
  'numbered', 'offered', 'offering', 'opened', 'openedTo', 'ordered', 'pacing', 'packed',
  'painted', 'painting', 'paired', 'parsed', 'picking', 'placed', 'pointing', 'pressed',
  'pressing', 'previewed', 'pulled', 'reaching', 'refusedFor', 'remainingWord', 'removed',
  'renamed', 'renamedIn', 'rendered', 'reordered', 'replaced', 'reported', 'resized',
  'resizing', 'resolved', 'rested', 'resting', 'rounded', 'routed', 'ruled', 'sampled',
  'saying', 'scrolled', 'scrolling', 'sealed', 'seated', 'selected', 'selectedIn', 'selecting',
  'separated', 'settled', 'showing', 'showingOf', 'sized', 'spanning', 'stacked', 'staged',
  'standing', 'standingIn', 'stated', 'stepped', 'stopped', 'stoppedWords', 'streamed',
  'styled', 'switched', 'talking', 'themed', 'thinking', 'tightened', 'titled', 'travelled',
  'turned', 'typed', 'typing', 'uncommented', 'uncounted', 'used', 'waiting', 'walked',
  'walking', 'watching', 'wearing', 'wheeled', 'widenedFor', 'worked', 'working',
  'workspacing', 'writing', 'zooming',
]

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
 * a ring is a thing.
 *
 * A third-person verb is not tested for. `carries` and `cells` end the same
 * way, and a factory here is free to take a plain noun, so no machine can tell
 * the narrator from the thing. A person reads those.
 */
export const verbal = (word) => /^.{2,}(ing|ed)$/.test(word)

/**
 * Whether a name is refused. The first word is what is read: it is the verb,
 * and the rest says what the verb acts on. `getSettings` asks for settings and
 * `onDropEntries` answers a drop; both end in a plural noun and neither is a
 * narrator. `carries` and `dressing` have nothing in front of the verb form,
 * so the verb form is the name.
 */
export function refused(name) {
  const first = words(name)[0] ?? ''
  if (!verbal(first)) return false
  return !Object.hasOwn(nouns, first)
}

/**
 * Every function one file declares: a `function`, and a `const` or `let` whose
 * value is an arrow or a function. A method is outside this — it is read as
 * part of the object it stands in.
 *
 * A parameter list is followed one bracket deep, which is as far as a default
 * value goes: `(deps = support())` is a signature and `(a) => (b) =>` is two.
 */
const DECLARED =
  /(?:^|\n)[ \t]*(?:export\s+)?(?:async\s+)?function\s+([a-z][\w$]*)|(?:^|\n)[ \t]*(?:export\s+)?(?:const|let)\s+([a-z][\w$]*)(?::[^=\n]*)?\s*=\s*(?:async\s+)?(?:\((?:[^()]|\([^()]*\))*\)|[a-z][\w$]*)\s*(?::[^=>\n]*)?=>/g

export function declares(source) {
  const found = []
  for (const one of code(source).matchAll(DECLARED)) found.push(one[1] ?? one[2])
  return found
}
