import type { CommandDeps, CommandTarget } from '@/shared/model/command'
import type { FileIcons } from '../model/rowIcons'
import type { Commands } from '../model/palette/palette'
import type { PickerSession } from '../model/picker/createPickerSession'
import type { ResultChoice, SearchState } from '../model/search/search'

export interface CommandPaletteProps {
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  places: FileIcons
  runChoice: (choice: ResultChoice) => void
  picker: PickerSession
}
