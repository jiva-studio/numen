/**
 * The ids of the commands the editor itself provides. Each is filed under the owner `app`, as
 * `app:<object>.<action>`; the commands of plugins are filed under the id of their plugin.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-host'

const formatAppCommandId = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  openNote: formatAppCommandId('note.open'),
  openNoteBeside: formatAppCommandId('note.openBeside'),
  goToNote: formatAppCommandId('note.goTo'),
  createNote: formatAppCommandId('note.create'),
  createChildNote: formatAppCommandId('note.createChild'),
  createParentNote: formatAppCommandId('note.createParent'),
  createJumpNote: formatAppCommandId('note.createJump'),
  renameNote: formatAppCommandId('note.rename'),
  trashNote: formatAppCommandId('note.trash'),
  deleteNote: formatAppCommandId('note.delete'),
  copyNotePath: formatAppCommandId('note.copyPath'),
  chooseTitleSync: formatAppCommandId('note.chooseTitleSync'),
  moveFile: formatAppCommandId('file.move'),
  createFolder: formatAppCommandId('folder.create'),
  closeTab: formatAppCommandId('tab.close'),
  searchVault: formatAppCommandId('vault.search'),
  openSettings: formatAppCommandId('settings.open'),
  chooseTheme: formatAppCommandId('appearance.chooseTheme'),
  chooseMode: formatAppCommandId('appearance.chooseMode'),
  chooseInterfaceSize: formatAppCommandId('appearance.chooseInterfaceSize'),
  chooseTextSize: formatAppCommandId('appearance.chooseTextSize'),
  choosePerformanceProfile: formatAppCommandId('performance.chooseProfile'),
  openVault: formatAppCommandId('vault.open'),
  createVault: formatAppCommandId('vault.create'),
  renameVault: formatAppCommandId('vault.rename'),
  removeVault: formatAppCommandId('vault.remove'),
  eraseVault: formatAppCommandId('vault.erase'),
} as const
