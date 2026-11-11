import { SLOT, type CommandTarget, type SlotHost } from '@numen/plugin-loader'

/** The entries the plugins place in the menu of a result of the search. */
export const createSearchMenu = (slots: Pick<SlotHost, 'getEntries'>) => ({
  getEntries: (target: CommandTarget) => slots.getEntries(SLOT.paletteSearchResult, target),
})
