/**
 * The ids of the commands the editor itself provides. Each is filed under the owner `app`, as
 * `app:<object>.<action>`; the commands of plugins are filed under the id of their plugin.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-host'

const formatAppCommandId = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  goToNote: formatAppCommandId('note.goTo'),
  createNote: formatAppCommandId('note.create'),
  createChildNote: formatAppCommandId('note.createChild'),
  createParentNote: formatAppCommandId('note.createParent'),
  createJumpNote: formatAppCommandId('note.createJump'),
  renameNote: formatAppCommandId('note.rename'),
  removeNote: formatAppCommandId('note.remove'),
  copyNotePath: formatAppCommandId('note.copyPath'),
  deleteText: formatAppCommandId('version.delete'),
  openFile: formatAppCommandId('file.open'),
  moveFile: formatAppCommandId('file.move'),
  createFolder: formatAppCommandId('folder.create'),
  closeTab: formatAppCommandId('tab.close'),
  detachTab: formatAppCommandId('tab.detach'),
  unlinkTab: formatAppCommandId('tab.unlink'),
  moveTabToWindow: formatAppCommandId('tab.moveToWindow'),
  linkTab: formatAppCommandId('tab.link'),
  searchVault: formatAppCommandId('vault.search'),
  openSettings: formatAppCommandId('settings.open'),
  openVault: formatAppCommandId('vault.open'),
  createVault: formatAppCommandId('vault.create'),
  renameVault: formatAppCommandId('vault.rename'),
  removeVault: formatAppCommandId('vault.remove'),
  eraseVault: formatAppCommandId('vault.erase'),
} as const

/** What the id of the command of a setting starts with: `setting:<owner>:<key>`. */
export const SETTING_COMMAND_PREFIX = 'setting:'

/**
 * The rows of the palette that run a command with an argument. A row is filed under its own key;
 * the command it runs and the argument it passes are declared on the row.
 */
export const ROW = {
  openNoteBeside: `${COMMAND.openFile}#beside`,
  openDocument: `${COMMAND.openFile}#document`,
  destroyNote: `${COMMAND.removeNote}#permanent`,
} as const

/** The names of the arguments of the command that opens a file and of the command that removes a note. */
export const ARGUMENT = {
  showing: 'showing',
  removal: 'removal',
} as const
