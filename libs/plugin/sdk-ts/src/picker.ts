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

// PickerHost is the list of items with check marks that a plugin opens in the window.
export interface PickerHost {
  // Opens the list and resolves when the person closes it.
  open(options: PickerOptions): Promise<void>
}
