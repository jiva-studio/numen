/**
 * The ids of the commands the editor itself provides. Each is filed under the owner `app`, as
 * `app:<object>.<action>`; the commands of plugins are filed under the id of their plugin.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-host'

const formatAppCommandId = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  createNote: formatAppCommandId('note.create'),
  createChildNote: formatAppCommandId('note.createChild'),
  createParentNote: formatAppCommandId('note.createParent'),
  createJumpNote: formatAppCommandId('note.createJump'),
  renameNote: formatAppCommandId('note.rename'),
  removeFile: formatAppCommandId('file.remove'),
  copyFilePath: formatAppCommandId('file.copyPath'),
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
  emptyTrash: formatAppCommandId('trash.empty'),
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
  goToNote: `${COMMAND.openFile}#note`,
} as const

/** The names of the arguments of the command that opens a file. */
export const ARGUMENT = {
  placement: 'placement',
} as const

/** The welcome row that puts the commands up. It is no command of its own, so it is its own identity. */
export const COMMANDS_ROW = 'commands'
