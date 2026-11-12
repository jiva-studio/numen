import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY } from '@numen/plugin-sdk'
import type { MountAccess } from '../../views'
import { checkPermission, type PermissionTarget } from './permission'

type ViewKind = typeof CONVERSATION | typeof NOTE_EDITOR | typeof VAULT_SUMMARY

/** What a view kind needs of the plugin that mounts it and of the plugin that draws it. */
interface ViewRule {
  readonly mounter: readonly string[]
  readonly registrant: readonly string[]
}

/**
 * The permissions of the two plugins of each view kind. The mounter hands the view a context and
 * the registrant receives it, so the registrant must hold everything the context lets a view do:
 * the conversation context carries an agent that asks and finishes (vault:read, vault:write), the
 * note editor reads and writes the note at its path, and the vault summary receives the name of a
 * vault. A new view kind states both lists here, an empty list when it needs none.
 */
const VIEW_RULES = new Map<string, ViewRule>([
  [CONVERSATION, { mounter: [], registrant: ['vault:read', 'vault:write'] }],
  [
    NOTE_EDITOR,
    { mounter: ['vault:read', 'vault:write'], registrant: ['vault:read', 'vault:write'] },
  ],
  [VAULT_SUMMARY, { mounter: [], registrant: ['vault:read'] }],
] satisfies [ViewKind, ViewRule][])

/**
 * Checks that the mounting plugin holds the permissions of the view kind and returns what the view
 * registry checks it with. A kind a plugin defines itself is not in the table: the registry then
 * requires the vault permissions of the plugin that registered it.
 */
export function requireMountAccess(capability: string, caller: PermissionTarget): MountAccess {
  const requirePerm = (permission: string, operation: string) =>
    checkPermission(caller, permission, operation)
  const rule = VIEW_RULES.get(capability)
  for (const permission of rule?.mounter ?? []) requirePerm(permission, 'mountView')
  return {
    inherits: rule === undefined,
    permissions: caller.manifest?.permissions ?? [],
    requirePerm,
  }
}

function getDeclaredViewCapabilities(caller: PermissionTarget): readonly string[] {
  const { manifest } = caller
  return manifest && 'contributes' in manifest ? (manifest.contributes?.viewCapabilities ?? []) : []
}

/**
 * Checks that a plugin may draw a view of the kind for others: its manifest lists the kind under
 * `contributes.viewCapabilities`, and it holds the permissions the kind hands the view. A kind a
 * plugin defines itself needs no more than the listing.
 */
export function requireViewRegistration(capability: string, caller: PermissionTarget): void {
  if (!getDeclaredViewCapabilities(caller).includes(capability)) {
    throw new Error(
      `Plugin "${caller.pluginId}" registers a view of "${capability}", which contributes.viewCapabilities does not list`,
    )
  }
  for (const permission of VIEW_RULES.get(capability)?.registrant ?? []) {
    checkPermission(caller, permission, 'register.view')
  }
}
