/**
 * A name typed over the one something carries, until it is committed or
 * abandoned.
 *
 * Why a name cannot be used is worked out once, from the names already taken,
 * and that one value is what the box says and what the commit consults.
 */
import { shallowRef } from 'vue'

/** A name being typed over the one something carries. */
interface Draft {
  /** What it is being typed over: a field by its name, a face by its identifier. */
  readonly over: string
  readonly text: string
}

/**
 * What naming something takes. `Objection` is what may be wrong with a name,
 * which is the rule's to say: a field is written in a slot and a face in a
 * heading, and the two are not written under the same rules.
 */
export interface NamingDeps<Objection> {
  /** The name the thing being typed over carries. */
  readonly carries: (over: string) => string
  /** The names already taken, which the one being typed is measured against. */
  readonly taken: (over: string) => readonly string[]
  /** Why a name cannot be used, and nothing where it can. */
  readonly amiss: (name: string, taken: readonly string[]) => Objection | null
  /** A name that may be used, committed. */
  readonly renamed: (over: string, name: string) => void
}

/** What naming answers: what a box holds, what is wrong with it, and the gestures. */
export interface NamingState<Objection> {
  /** What is in the box: the name it carries, or what is being typed over it. */
  readonly text: (over: string) => string
  /** Why what is in the box cannot be used, and nothing while it can. */
  readonly objection: (over: string) => Objection | null
  /** Something was typed into the box. */
  readonly setDraft: (over: string, text: string) => void
  /** What was typed is committed, and nothing where it objects or says what it said. */
  readonly commit: (over: string) => void
  /** A key struck in the box: a break commits what was typed, escape abandons it. */
  readonly onKey: (press: KeyboardEvent, over: string) => void
}

export function useNaming<Objection>(deps: NamingDeps<Objection>): NamingState<Objection> {
  /** What is being typed, over the thing it is being typed over. */
  const draft = shallowRef<Draft | null>(null)

  /** What is being typed over this thing, and nothing where nothing is. */
  const getDraft = (over: string): string | null => {
    const held = draft.value
    return held?.over === over ? held.text : null
  }

  const text = (over: string): string => getDraft(over) ?? deps.carries(over)

  const objection = (over: string): Objection | null => {
    const said = getDraft(over)
    return said === null ? null : deps.amiss(said, deps.taken(over))
  }

  const setDraft = (over: string, text: string): void => {
    draft.value = { over, text }
  }

  const commit = (over: string): void => {
    const said = getDraft(over)
    // The objection stands on what is being typed, so it is read while it is.
    const amiss = objection(over)
    draft.value = null
    if (said === null) return

    const name = said.trim()
    if (name === deps.carries(over)) return
    if (amiss !== null) return
    deps.renamed(over, name)
  }

  const onKey = (press: KeyboardEvent, over: string): void => {
    if (press.key === 'Enter') {
      press.preventDefault()
      commit(over)
      ;(press.currentTarget as HTMLInputElement).blur()
      return
    }
    if (press.key === 'Escape') {
      press.preventDefault()
      draft.value = null
    }
  }

  return { text, objection, setDraft, commit, onKey }
}
