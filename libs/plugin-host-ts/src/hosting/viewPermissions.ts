import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY } from '@numen/plugin-sdk'

type ViewKind = typeof CONVERSATION | typeof NOTE_EDITOR | typeof VAULT_SUMMARY

/**
 * The permissions the plugin that mounts a view of each kind must hold. A new view kind states its
 * requirement here, an empty list when it needs none.
 */
const VIEW_PERMISSIONS = new Map<string, readonly string[]>([
  [CONVERSATION, []],
  [NOTE_EDITOR, ['vault:read']],
  [VAULT_SUMMARY, []],
] satisfies [ViewKind, readonly string[]][])

/**
 * The permissions mounting a view of the capability needs, or undefined for a capability a plugin
 * defines itself, which needs the vault permissions of the plugin that registered it.
 */
export function getViewPermissions(capability: string): readonly string[] | undefined {
  return VIEW_PERMISSIONS.get(capability)
}
