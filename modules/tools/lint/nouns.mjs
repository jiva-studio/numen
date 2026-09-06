/**
 * A type is a noun or a noun phrase — not a gerund, not a participle.
 *
 * A stranger meeting `Filing` or `Drawn` cold cannot say what the thing is or
 * which part of the application it belongs to, and the same word came to name
 * a flashcards pill, an editor result and a handler at once. Verb phrases name
 * functions; nouns name the things they act on.
 *
 * A machine reading a name can only see how it ends, and a suffix test cannot
 * tell a gerund from a word that merely ends in those letters. So the words
 * that are not gerunds are written down, with what each one means. It is a
 * dictionary, not a list of exemptions: a reader can apply the rule without
 * reading it, and a word that has left the source has to leave the dictionary.
 */

/** The words of ours ending in -ing or -ed that are ordinary English nouns. */
export const nouns = {
  binding: 'a key bound to a command, which is CodeMirror\'s own word',
  deferred: "an answer handed over later; the field's name for this object",
  drawing: 'a picture',
  heading: 'a line a section of a note stands under',
  landing: 'where something let go comes to rest',
  opening: 'the way in, and how wide it is',
  recording: 'a sound file',
  routing: 'the way a line is taken from one place to another',
  seating: 'where the nodes of a plex are put',
  spacing: 'the room left between things',
  timing: 'when something happens, and how long it takes',
}

/**
 * owed are the types still named by a gerund, and the list only shrinks.
 *
 * PlexShowing is the last of them because the stem runs through SHOWINGS,
 * ShowingDescriptor and showingOf, and renaming the type alone would leave the
 * family behind. That is one decision, not five.
 */
export const owed = ['PlexShowing']

/** The words of one name, as a reader says them. */
export const words = (name) =>
  [...name.matchAll(/[A-Z]+(?![a-z])|[A-Z][a-z0-9]*|^[a-z][a-z0-9]*/g)].map((one) =>
    one[0].toLowerCase(),
  )

/** Whether a name reads as a gerund or a participle. */
export function refused(name) {
  if (!/(ing|ed)$/.test(name)) return false
  const said = words(name)
  return !Object.hasOwn(nouns, said[said.length - 1] ?? '')
}

/** Every type, interface, class and enum one file declares. */
const DECLARED = /^[ \t]*(?:export\s+)?(?:declare\s+)?(?:abstract\s+)?(?:type|interface|class|enum)\s+([A-Za-z_$][\w$]*)/gm

export function declares(source) {
  return [...source.matchAll(DECLARED)].map((one) => one[1])
}
