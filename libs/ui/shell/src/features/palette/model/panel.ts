/**
 * The panel of actions standing over the foot of the palette.
 *
 * It is about the item that was lit when it opened, and an item that stops
 */
import { ref, watch, type Ref } from 'vue'
import type { PaletteAction } from '../lib/item'

export interface ActionPanelState {
  readonly open: Ref<boolean>
  /** An action chosen in the panel, on the item it was opened about. */
  readonly chooseAction: (action: string) => void
  /** A press anywhere but on the panel puts the panel away. */
  readonly onPress: () => void
  /** A press on the backdrop closes the panel and the palette beneath it. */
  readonly onBackdrop: () => void
}

export interface ActionPanelOptions {
  readonly getActions: () => readonly PaletteAction[]
  /** The item the panel is about, by its identity. */
  readonly getLit: () => string | undefined
  /** An item was chosen with an action. */
  readonly choose: (item: string, action: string) => void
  /** Requests that the palette close. */
  readonly dismiss: () => void
}

export function useActionPanel(options: ActionPanelOptions): ActionPanelState {
  const open = ref(false)

  watch(options.getActions, (now) => {
    if (!now.length) open.value = false
  })

  const chooseAction = (action: string): void => {
    const item = options.getLit()
    if (item) options.choose(item, action)
  }

  const onPress = (): void => {
    open.value = false
  }

  const onBackdrop = (): void => {
    if (open.value) return void (open.value = false)
    options.dismiss()
  }

  return { open, chooseAction, onPress, onBackdrop }
}
