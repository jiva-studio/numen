import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY } from '@numen/plugin-sdk'
import type { MountAccess } from '../../views'
import { checkPermission, type PermissionTarget } from './permission'

type ViewKind = typeof CONVERSATION | typeof NOTE_EDITOR | typeof VAULT_SUMMARY

/**
 * The permissions the plugin that mounts a view of each kind must hold. A new view kind states its
 * requirement here, an empty list when it needs none.
 */
const VIEW_PERMISSIONS = new Map<string, readonly string[]>([
  [CONVERSATION, []],
  [NOTE_EDITOR, ['vault:read', 'vault:write']],
  [VAULT_SUMMARY, []],
] satisfies [ViewKind, readonly string[]][])

/**
 * Checks that the mounting plugin holds the permissions of the view kind and returns what the view
 * registry checks it with. A kind a plugin defines itself is not in the table: the registry then
 * requires the vault permissions of the plugin that registered it.
 */
export function requireMountAccess(capability: string, caller: PermissionTarget): MountAccess {
  const requirePerm = (permission: string, operation: string) =>
    checkPermission(caller, permission, operation)
  const required = VIEW_PERMISSIONS.get(capability)
  for (const permission of required ?? []) requirePerm(permission, 'mountView')
  return {
    inherits: required === undefined,
    permissions: caller.manifest?.permissions ?? [],
    requirePerm,
  }
}
