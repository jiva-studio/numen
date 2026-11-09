import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY } from '@numen/plugin-sdk'

type ViewKind = typeof CONVERSATION | typeof NOTE_EDITOR | typeof VAULT_SUMMARY

/**
 * The permissions the plugin that mounts a view of each kind must hold. A new view kind states its
 * requirement here, an empty list when it needs none.
 */
const VIEW_PERMISSIONS = {
  [CONVERSATION]: [],
  [NOTE_EDITOR]: ['vault:read'],
  [VAULT_SUMMARY]: [],
} as const satisfies Record<ViewKind, readonly string[]>

/** The permissions mounting a view of the capability needs; a capability of a plugin needs none. */
export function getViewPermissions(capability: string): readonly string[] {
  return (VIEW_PERMISSIONS as Record<string, readonly string[]>)[capability] ?? []
}
