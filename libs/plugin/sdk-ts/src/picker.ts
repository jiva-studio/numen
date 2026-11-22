// PickerItem is one row of the picker. Its state is set, mixed (on for some of what it stands for) or unset.
export interface PickerItem {
  readonly id: string
  readonly title: string
  readonly state: 'set' | 'mixed' | 'unset'
  readonly color?: string
}

// PickerOptions is the list a plugin asks the person to pick from. Rows with a state of set or mixed come
// first, then the rows of recent (ids, most recent first), then the rest by title. The create row, named by
// getCreateTitle, shows while the typed text is not empty and no item has that title. onToggle is called at
// once for each toggle and resolves with the item as it is after the change; onCreate resolves with the new
// item, already set.
export interface PickerOptions {
  readonly title: string
  readonly placeholder: string
  readonly items: readonly PickerItem[]
  readonly recent: readonly string[]
  readonly getCreateTitle: (text: string) => string
  readonly getCreateColor?: (text: string) => string
  readonly onToggle: (id: string) => Promise<PickerItem>
  readonly onCreate: (text: string) => Promise<PickerItem>
}

// PromptTextOptions is a line of text a plugin asks the person for. The title names what is asked, and value is
// the text the line starts with.
export interface PromptTextOptions {
  readonly title: string
  readonly value?: string
}

// ChoiceOption is one answer of a choice: the value the plugin gets back and the title the person reads.
export interface ChoiceOption {
  readonly value: string
  readonly title: string
}

// SelectOptionOptions is the one answer a plugin asks the person to pick among options, listed in the order given.
export interface SelectOptionOptions {
  readonly title: string
  readonly options: readonly ChoiceOption[]
}

// PickerHost is what a plugin asks of the person in the palette of the window.
export interface PickerHost {
  // Opens the list and resolves when the person closes it.
  open(options: PickerOptions): Promise<void>
  // Asks for a line of text and resolves with it, trimmed, or with undefined when the person dismisses it.
  promptText(options: PromptTextOptions): Promise<string | undefined>
  // Asks for one of the options and resolves with its value, or with undefined when the person dismisses it.
  selectOption(options: SelectOptionOptions): Promise<string | undefined>
}
