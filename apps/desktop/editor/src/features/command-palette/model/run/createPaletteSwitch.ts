import type { Commands } from '../palette/palette'
import type { SearchState } from '../search/search'

/** The two palettes of the window, of which one is open at a time. */
export function createPaletteSwitch(
  commands: Pick<Commands, 'palette'>,
  search: Pick<SearchState, 'open' | 'setOpen'>,
) {
  return {
    /** Whether the search or the commands are open. */
    isOpen: (): boolean => commands.palette.open.value || search.open.value,
    /** Closes the commands and opens the search, or closes the search where it is open. */
    toggleSearch: (): void => {
      commands.palette.setOpen(false)
      search.setOpen(!search.open.value)
    },
    /** Closes the search and opens the commands, or closes the commands where they are open. */
    toggleCommands: (): void => {
      search.setOpen(false)
      commands.palette.setOpen(!commands.palette.open.value)
    },
    /** Closes the search. */
    closeSearch: (): void => search.setOpen(false),
    /** Closes the commands and opens the search. */
    openSearch: (): void => {
      commands.palette.setOpen(false)
      search.setOpen(true)
    },
  }
}
