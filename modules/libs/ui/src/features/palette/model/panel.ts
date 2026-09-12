/**
 * The panel of actions standing over the foot of the palette.
 *
 * It is about the item that was lit when it opened, and an item that stops
 * offering anything leaves it about nothing.
 */
import { ref, watch, type Ref } from 'vue'
import type { PaletteAction } from '../lib/item'

export interface ActionPanelState {
  /** Whether the panel stands over the palette. */
  readonly open: Ref<boolean>
  /** An action chosen in the panel, on the item it was opened about. */
  readonly chooseAction: (action: string) => void
  /** A press anywhere but on the panel puts the panel away. */
  readonly onPress: () => void
  /** A press on the ground: the panel goes, and the palette under it. */
  readonly onGround: () => void
}

export function useActionPanel(
  getActions: () => readonly PaletteAction[],
  /** The item the panel is about, by its identity. */
  getLit: () => string | undefined,
  /** An item was chosen, and what was asked of it. */
  choose: (item: string, action: string) => void,
  /** The palette asks to be put away. */
  dismiss: () => void,
): ActionPanelState {
  const open = ref(false)

  watch(getActions, (now) => {
    if (!now.length) open.value = false
  })

  const chooseAction = (action: string): void => {
    const item = getLit()
    if (item) choose(item, action)
  }

  const onPress = (): void => {
    open.value = false
  }

  const onGround = (): void => {
    if (open.value) return void (open.value = false)
    dismiss()
  }

  return { open, chooseAction, onPress, onGround }
}
