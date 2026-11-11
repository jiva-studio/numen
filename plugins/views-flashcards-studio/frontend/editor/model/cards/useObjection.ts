import { computed, type ComputedRef } from 'vue'
import type { Objection } from '../../../lib/cards'
import type { NameEntryState } from './nameEntry'

/** Why what is in the name box of a target cannot be used, and null while it can. */
export function useObjection<Why extends Objection>(
  getNaming: () => NameEntryState<Why>,
  getTargetId: () => string,
): ComputedRef<Why | null> {
  return computed(() => getNaming().getObjection(getTargetId()))
}

/** The message for an objection, null while there is none. The formatter is read when the message is, so it follows the caller's words. */
export function useObjectionMessage<Why extends Objection>(
  objection: Readonly<{ value: Why | null }>,
  getFormat: () => (objection: Why) => string,
): ComputedRef<string | null> {
  return computed(() => {
    const why = objection.value
    return why === null ? null : getFormat()(why)
  })
}
