/**
 * In-progress name editing for items identified by target ID.
 */
import { shallowRef } from 'vue'
import type { NameCheckResult, Objection } from '../lib/order'

/** In-progress name draft. */
interface Draft {
  readonly targetId: string
  readonly text: string
}

export interface NameEntryDeps<Why extends Objection> {
  readonly getName: (targetId: string) => string
  readonly getTakenNames: (targetId: string) => readonly string[]
  readonly checkName: (name: string, acquired: readonly string[]) => NameCheckResult<Why>
  readonly rename: (targetId: string, name: string) => void
}

export interface NameEntryState<Why extends Objection> {
  readonly getText: (targetId: string) => string
  readonly getObjection: (targetId: string) => Why | null
  readonly setDraft: (targetId: string, text: string) => void
  readonly commit: (targetId: string) => void
  readonly onKey: (event: KeyboardEvent, targetId: string) => void
}

export function useNameEntry<Why extends Objection>(deps: NameEntryDeps<Why>): NameEntryState<Why> {
  const draft = shallowRef<Draft | null>(null)

  const getDraft = (targetId: string): string | null => {
    const current = draft.value
    return current?.targetId === targetId ? current.text : null
  }

  const getText = (targetId: string): string => getDraft(targetId) ?? deps.getName(targetId)

  const check = (targetId: string): NameCheckResult<Why> | null => {
    const drafted = getDraft(targetId)
    return drafted === null ? null : deps.checkName(drafted, deps.getTakenNames(targetId))
  }

  const getObjection = (targetId: string): Why | null => check(targetId)?.objection ?? null

  const setDraft = (targetId: string, text: string): void => {
    draft.value = { targetId, text }
  }

  const commit = (targetId: string): void => {
    const checked = check(targetId)
    draft.value = null
    if (checked === null || checked.objection !== null) return
    if (checked.name === deps.getName(targetId)) return
    deps.rename(targetId, checked.name)
  }

  const onKey = (event: KeyboardEvent, targetId: string): void => {
    if (event.key === 'Enter') {
      event.preventDefault()
      commit(targetId)
      ;(event.currentTarget as HTMLInputElement).blur()
      return
    }
    if (event.key === 'Escape') {
      event.preventDefault()
      draft.value = null
    }
  }

  return { getText, getObjection, setDraft, commit, onKey }
}
