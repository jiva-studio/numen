import type { AskOptions, ChooseOptions, PickerHost, PickerItem, PickerOptions } from '../picker'

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
  // The question asked and not yet answered.
  getQuestion(): AskOptions | ChooseOptions | undefined
  // Answers the question asked, as the person who types or picks a value.
  answer(value: string): void
  // Dismisses the question asked, which resolves it with undefined.
  dismiss(): void
  // Closes the list, which settles the open call.
  close(): void
}

// createMockPickerHost builds a picker host that keeps the opened list in memory.
export function createMockPickerHost(): MockPickerHost {
  let opened: PickerOptions | undefined
  let items: readonly PickerItem[] = []
  let settle: (() => void) | undefined
  let question:
    | { options: AskOptions | ChooseOptions; resolve: (value: string | undefined) => void }
    | undefined

  const settleQuestion = (value: string | undefined): void => {
    const current = question
    question = undefined
    current?.resolve(value)
  }

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
    ask: (options) =>
      new Promise((resolve) => {
        question = { options, resolve }
      }),
    choose: (options) =>
      new Promise((resolve) => {
        question = { options, resolve }
      }),
    getQuestion: () => question?.options,
    answer: (value) => settleQuestion(value),
    dismiss: () => settleQuestion(undefined),
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
