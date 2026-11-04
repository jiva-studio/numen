/**
 * The ids of the commands the editor itself provides. Each is filed under the owner `app`, as
 * `app:<object>.<action>`; the commands of plugins are filed under the id of their plugin.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-host'

const own = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  openNote: own('note.open'),
  openNoteBeside: own('note.open-beside'),
  goToStartNote: own('note.go-to-start'),
  goToNote: own('note.go-to'),
  createNote: own('note.create'),
  createChildNote: own('note.create-child'),
  createParentNote: own('note.create-parent'),
  createJumpNote: own('note.create-jump'),
  renameNote: own('note.rename'),
  trashNote: own('note.trash'),
  deleteNote: own('note.delete'),
  copyNotePath: own('note.copy-path'),
  chooseTitleSync: own('note.choose-title-sync'),
  moveFile: own('file.move'),
  createFolder: own('folder.create'),
  closeTab: own('tab.close'),
  searchVault: own('vault.search'),
  openSettings: own('settings.open'),
  chooseTheme: own('appearance.choose-theme'),
  chooseMode: own('appearance.choose-mode'),
  chooseInterfaceSize: own('appearance.choose-interface-size'),
  chooseTextSize: own('appearance.choose-text-size'),
  choosePerformanceProfile: own('performance.choose-profile'),
  openVault: own('vault.open'),
  createVault: own('vault.create'),
  renameVault: own('vault.rename'),
  removeVault: own('vault.remove'),
  eraseVault: own('vault.erase'),
} as const
