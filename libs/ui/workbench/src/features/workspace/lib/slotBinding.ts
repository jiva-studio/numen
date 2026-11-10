import type { TabId } from './node'

/** What a tab slot is bound to. */
export interface SlotBinding {
  readonly id: TabId
  readonly mark: string
}

/** The binding a slot received, and an empty one where it received none. */
export const getSlotProps = (bound: Partial<SlotBinding> | undefined): SlotBinding =>
  (bound ?? {}) as SlotBinding
