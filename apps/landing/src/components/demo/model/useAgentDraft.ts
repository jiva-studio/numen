import { computed, type WritableComputedRef } from 'vue'
import type { Agent } from './agent'

/** The text of the question being written, two-way: reading gives the agent's draft and writing sets it. */
export function useAgentDraft(getAgent: () => Agent): WritableComputedRef<string> {
  return computed({
    get: () => getAgent().draft.value,
    set: (text: string) => getAgent().setDraft(text),
  })
}
