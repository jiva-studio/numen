/**
 * What the window has said, and for how long it holds on to it.
 *
 * Every part of the window that answers a person speaks here under a name of
 * its own, and the corner draws what stands. Nothing here knows how any of it
 * is drawn.
 */
import { ref, type Ref } from 'vue'

/**
 * What kind of thing the window said.
 *
 * A refusal is what did not happen. A caution is worth reading and needs
 * nothing done at once. A report is what a command did. A state is so until
 * something else makes it not so.
 */
export type Kind = 'refusal' | 'caution' | 'report' | 'state'

/** One thing the window has said. */
export interface Told {
  /** What this utterance is addressed by, which no two of them share. */
  readonly id: string
  /** Who said it. A second word under one name replaces the first. */
  readonly name: string
  readonly kind: Kind
  readonly says: string
}

/** One part of the window speaking. Nothing said clears what it last said. */
export type Voice = (text: string, kind?: Kind) => void

/** What the window has said, and what changes it. */
export interface Telling {
  readonly said: Ref<readonly Told[]>
  /** A voice under a name of its own. */
  under(name: string): Voice
  /** A word the person is finished with, by the identity it was given. */
  forget(id: string): void
}

/**
 * Every word gets an identity of its own.
 *
 * What draws these remembers by identity when each arrived and which the person
 * put away, so a second word under one name is a second word and is read as
 * one.
 */
export function telling(): Telling {
  const said = ref<readonly Told[]>([])
  let minted = 0

  const under =
    (name: string): Voice =>
    (text, kind = 'report') => {
      // A voice saying again what it is already saying has said nothing new,
      // and what stands keeps its identity and its place.
      const stood = said.value.find((one) => one.name === name)
      if (stood === undefined && text === '') return
      if (stood?.says === text && stood.kind === kind) return

      const rest = said.value.filter((one) => one.name !== name)
      if (text === '') {
        said.value = rest
        return
      }
      minted += 1
      said.value = [...rest, { id: `${name}#${minted}`, name, kind, says: text }]
    }

  const forget = (id: string): void => {
    said.value = said.value.filter((one) => one.id !== id)
  }

  return { said, under, forget }
}
