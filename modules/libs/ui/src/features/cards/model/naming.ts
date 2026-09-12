/**
 * A name typed over the one something carries, until it is committed or
 * abandoned.
 *
 * Why a name cannot be used is worked out once, from the names already taken,
 * and that one value is what the box says and what the commit consults.
 */
import { shallowRef } from 'vue'
import type { NameCheckResult, Objection } from '../lib/order'

/** A name being typed over the one something carries. */
interface Draft {
  /** What it is being typed over: a field by its name, a face by its identifier. */
  readonly over: string
  readonly text: string
}

/**
 * What naming something takes. `Why` is the objections a name may draw, which
 * is the rule's to say: a field is written in a slot and a face in a heading,
 * and the two are not written under the same rules.
 */
export interface NamingDeps<Why extends Objection> {
  /** The name the thing being typed over carries. */
  readonly getName: (over: string) => string
  /** The names already taken, which the one being typed is measured against. */
  readonly getTakenNames: (over: string) => readonly string[]
  /** The name as it would be written, and why it cannot be used. */
  readonly checkName: (name: string, taken: readonly string[]) => NameCheckResult<Why>
  /** A name that may be used, committed. */
  readonly rename: (over: string, name: string) => void
}

/** What naming answers: what a box holds, what is wrong with it, and the gestures. */
export interface NamingState<Why extends Objection> {
  /** What is in the box: the name it carries, or what is being typed over it. */
  readonly getText: (over: string) => string
  /** Why what is in the box cannot be used, and nothing while it can. */
  readonly getObjection: (over: string) => Why | null
  /** Something was typed into the box. */
  readonly setDraft: (over: string, text: string) => void
  /** What was typed is committed, and nothing where it objects or says what it said. */
  readonly commit: (over: string) => void
  /** A key struck in the box: a break commits what was typed, escape abandons it. */
  readonly onKey: (press: KeyboardEvent, over: string) => void
}

export function useNaming<Why extends Objection>(deps: NamingDeps<Why>): NamingState<Why> {
  /** What is being typed, over the thing it is being typed over. */
  const draft = shallowRef<Draft | null>(null)

  /** What is being typed over this thing, and nothing where nothing is. */
  const getDraft = (over: string): string | null => {
    const held = draft.value
    return held?.over === over ? held.text : null
  }

  const getText = (over: string): string => getDraft(over) ?? deps.getName(over)

  const check = (over: string): NameCheckResult<Why> | null => {
    const said = getDraft(over)
    return said === null ? null : deps.checkName(said, deps.getTakenNames(over))
  }

  const getObjection = (over: string): Why | null => check(over)?.objection ?? null

  const setDraft = (over: string, text: string): void => {
    draft.value = { over, text }
  }

  const commit = (over: string): void => {
    // The check stands on what is being typed, so it is read while it is.
    const checked = check(over)
    draft.value = null
    if (checked === null || checked.objection !== null) return
    if (checked.name === deps.getName(over)) return
    deps.rename(over, checked.name)
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

  return { getText, getObjection, setDraft, commit, onKey }
}
