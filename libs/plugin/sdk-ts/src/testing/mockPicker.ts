import type { PickerHost, PickerItem, PickerOptions } from '../picker'

// MockPickerHost is a PickerHost that holds the list a plugin opened, so a test acts as the person
// who toggles its rows and closes it.
export interface MockPickerHost extends PickerHost {
  // The options of the list opened last.
  getOpened(): PickerOptions | undefined
  // The rows as the person sees them now.
  getItems(): readonly PickerItem[]
  // The title of the row that creates an item from the text typed.
  getCreateTitle(text: string): string
  // The colour of the row that creates an item from the text typed.
  getCreateColor(text: string): string | undefined
  // Toggles a row: asks the plugin, then shows the item it answers with.
  toggle(id: string): Promise<void>
  // Chooses the create row for a text: asks the plugin, then adds the item it answers with.
  create(text: string): Promise<void>
  // Closes the list, which settles the open call.
  close(): void
}

// createMockPickerHost builds a picker host that keeps the opened list in memory.
export function createMockPickerHost(): MockPickerHost {
  let opened: PickerOptions | undefined
  let items: readonly PickerItem[] = []
  let settle: (() => void) | undefined

  const read = (): PickerOptions => {
    if (!opened) throw new Error('no list is open')
    return opened
  }

  return {
    open: (options) => {
      opened = options
      items = options.items
      return new Promise<void>((resolve) => {
        settle = resolve
      })
    },
    getOpened: () => opened,
    getItems: () => items,
    getCreateTitle: (text) => read().getCreateTitle(text),
    getCreateColor: (text) => read().getCreateColor?.(text),
    toggle: async (id) => {
      const item = await read().onToggle(id)
      items = items.map((one) => (one.id === item.id ? item : one))
    },
    create: async (text) => {
      const item = await read().onCreate(text)
      items = [...items, item]
    },
    close: () => settle?.(),
  }
}
