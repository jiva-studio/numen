/**
 * The id a command is filed under in the registries of a window: the owner, a colon, and the id the
 * owner gave the command. The owner is the id of the plugin that provides the command, or `app` for
 * a command of the application itself. Plugin ids contain dots and no colon, so the first colon
 * ends the owner.
 */

/** The owner of the commands the application itself provides. */
export const APP_COMMAND_OWNER = 'app'

/** Returns the full id of the command `id` of `owner`. */
export function formatCommandId(owner: string, id: string): string {
  return `${owner}:${id}`
}

/** Returns the owner a full command id starts with, and an empty string where it has none. */
export function getCommandOwner(fullId: string): string {
  const colon = fullId.indexOf(':')
  return colon < 0 ? '' : fullId.slice(0, colon)
}
