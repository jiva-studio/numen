/**
 * The Cmd/Ctrl+drag of the bar entries. The contract of `.agents/tasks/1803-slot-layout/spec.md`
 * §1.10, declared so its acceptance stories load; the body is not written yet.
 */
import type { BarSlotName } from '@numen/plugin-sdk'
import type { SlotLayout, SlotLayoutChange } from '@numen/plugin-host'

export interface SlotDragDeps {
  readonly isMac: () => boolean
  readonly findLayoutKey: (itemKey: string) => string | undefined
  readonly listLayoutKeys: (slot: BarSlotName) => readonly string[]
  readonly getLayout: () => SlotLayout
  readonly writeLayout: (change: SlotLayoutChange) => Promise<void>
}

export function createSlotDrag(_deps: SlotDragDeps): never {
  throw new Error('createSlotDrag is not implemented')
}
