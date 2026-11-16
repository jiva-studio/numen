// PickerState is whether an item of the picker is on, on for some of what it stands for, or off.
export type PickerState = 'set' | 'mixed' | 'unset'

// PickerItem is one row of the picker.
export interface PickerItem {
  readonly id: string
  readonly title: string
  readonly state: PickerState
}

// PickerOptions is the list a plugin asks the person to pick from. The rows with a state of set or mixed
// stand first, then the rows of recent, then the rest by title.
export interface PickerOptions {
  // The title of the list.
  readonly title: string
  // The placeholder of the filter field.
  readonly placeholder: string
  readonly items: readonly PickerItem[]
  // The ids of the items used lately, the most recent first.
  readonly recent: readonly string[]
  // The title of the row that creates an item from the text typed, shown while no item has that title.
  readonly getCreateTitle: (text: string) => string
  // Called at once for each toggle; resolves with the item as it is after the change.
  readonly onToggle: (id: string) => Promise<PickerItem>
  // Called for the row that creates an item; resolves with the new item, already set.
  readonly onCreate: (text: string) => Promise<PickerItem>
}

// PickerHost is the list of items with check marks that a plugin opens in the window.
export interface PickerHost {
  // Opens the list and resolves when the person closes it.
  open(options: PickerOptions): Promise<void>
}
