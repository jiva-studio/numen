import { useSlotMenu, type Caught, type CommandItem, type SlotMenuState, type SlotMenuWords } from '@numen/plugin-sdk'

/** The target of a bar entry; the bar passes none of its own, and the window runs a bar entry over the front tab. */
export const BAR = 'bar'

export interface BarSlotMenuSources {
  readonly getListItems: () => (command: string) => Promise<readonly CommandItem[]>
  readonly getReportError: () => ((error: Caught) => void) | undefined
  readonly words: SlotMenuWords
}

/** The slot menu of a bar button, reading the items of a command and reporting a failed read through what the caller holds now. */
export function useBarSlotMenu({
  getListItems,
  getReportError,
  words,
}: BarSlotMenuSources): SlotMenuState<typeof BAR> {
  return useSlotMenu<typeof BAR>({
    listItems: (command) => getListItems()(command),
    words,
    onError: (error) => getReportError()?.(error),
  })
}
