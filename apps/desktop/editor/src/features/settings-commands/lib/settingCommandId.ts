import { SETTING_COMMAND_PREFIX } from '@/shared/model/command'

/** The id of the command of a setting: `setting:<owner>:<key>`, the key a path joined by dots. */
export const getSettingCommandId = (owner: string, key: string): string =>
  `${SETTING_COMMAND_PREFIX}${owner}:${key}`

/** The owner and the key a command id names, and nothing for an id of another command. */
export const parseSettingCommandId = (
  id: string,
): { readonly owner: string; readonly key: string } | null => {
  const match = /^setting:([^:]+):(.+)$/.exec(id)
  return match?.[1] && match[2] ? { owner: match[1], key: match[2] } : null
}
