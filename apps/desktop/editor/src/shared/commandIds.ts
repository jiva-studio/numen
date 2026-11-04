/**
 * The ids of the commands the editor itself provides. Each is filed under the owner `app`, as
 * `app:<object>.<action>`; the commands of plugins are filed under the id of their plugin.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-host'

const formatAppCommandId = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  openNote: formatAppCommandId('note.open'),
  openNoteBeside: formatAppCommandId('note.open-beside'),
  goToStartNote: formatAppCommandId('note.go-to-start'),
  goToNote: formatAppCommandId('note.go-to'),
  createNote: formatAppCommandId('note.create'),
  createChildNote: formatAppCommandId('note.create-child'),
  createParentNote: formatAppCommandId('note.create-parent'),
  createJumpNote: formatAppCommandId('note.create-jump'),
  renameNote: formatAppCommandId('note.rename'),
  trashNote: formatAppCommandId('note.trash'),
  deleteNote: formatAppCommandId('note.delete'),
  copyNotePath: formatAppCommandId('note.copy-path'),
  chooseTitleSync: formatAppCommandId('note.choose-title-sync'),
  moveFile: formatAppCommandId('file.move'),
  createFolder: formatAppCommandId('folder.create'),
  closeTab: formatAppCommandId('tab.close'),
  searchVault: formatAppCommandId('vault.search'),
  openSettings: formatAppCommandId('settings.open'),
  chooseTheme: formatAppCommandId('appearance.choose-theme'),
  chooseMode: formatAppCommandId('appearance.choose-mode'),
  chooseInterfaceSize: formatAppCommandId('appearance.choose-interface-size'),
  chooseTextSize: formatAppCommandId('appearance.choose-text-size'),
  choosePerformanceProfile: formatAppCommandId('performance.choose-profile'),
  openVault: formatAppCommandId('vault.open'),
  createVault: formatAppCommandId('vault.create'),
  renameVault: formatAppCommandId('vault.rename'),
  removeVault: formatAppCommandId('vault.remove'),
  eraseVault: formatAppCommandId('vault.erase'),
} as const
