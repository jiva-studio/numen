/**
 * The ids of the commands the editor itself provides that no feature has taken over yet. Each is
 * filed under the owner `app`, as `app:<object>.<action>`.
 */
import { APP_COMMAND_OWNER, formatCommandId } from '@numen/plugin-loader'

const formatAppCommandId = (id: string): string => formatCommandId(APP_COMMAND_OWNER, id)

export const COMMAND = {
  deleteText: formatAppCommandId('version.delete'),
  closeTab: formatAppCommandId('tab.close'),
  detachTab: formatAppCommandId('tab.detach'),
  unlinkTab: formatAppCommandId('tab.unlink'),
  moveTabToWindow: formatAppCommandId('tab.moveToWindow'),
  linkTab: formatAppCommandId('tab.link'),
} as const

/** What the id of the command of a setting starts with: `setting:<owner>:<key>`. */
export const SETTING_COMMAND_PREFIX = 'setting:'
