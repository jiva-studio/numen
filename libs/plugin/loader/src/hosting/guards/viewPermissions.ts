import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY } from '@numen/plugin-sdk'
import type { MountAccess } from '../../views'
import { checkPermission, type PermissionTarget } from './permission'

type ViewKind = typeof CONVERSATION | typeof NOTE_EDITOR | typeof VAULT_SUMMARY

/** What a view kind needs of the plugin that mounts it and of the plugin that draws it. */
interface ViewRule {
  readonly owner: string
  readonly mounter: readonly string[]
  readonly registrant: readonly string[]
}

/** Each built-in view has one provider and separate permission requirements for registration and mounting. */
const VIEW_RULES = new Map<string, ViewRule>([
  [CONVERSATION, { owner: 'md.numen.views.agent.chat', mounter: [], registrant: [] }],
  [
    NOTE_EDITOR,
    {
      owner: 'md.numen.notes',
      mounter: ['vault:read', 'vault:write'],
      registrant: ['vault:read', 'vault:write'],
    },
  ],
  [
    VAULT_SUMMARY,
    { owner: 'md.numen.views.flashcards.studio', mounter: [], registrant: ['vault:read'] },
  ],
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

/** Checks the manifest declaration and, for a built-in view, the provider identity and permissions. */
export function requireViewRegistration(capability: string, caller: PermissionTarget): void {
  if (!getDeclaredViewCapabilities(caller).includes(capability)) {
    throw new Error(
      `Plugin "${caller.pluginId}" registers a view of "${capability}", which contributes.viewCapabilities does not list`,
    )
  }
  const rule = VIEW_RULES.get(capability)
  if (rule && caller.pluginId !== rule.owner) {
    throw new Error(
      `Plugin "${caller.pluginId}" cannot register "${capability}", owned by "${rule.owner}"`,
    )
  }
  for (const permission of rule?.registrant ?? []) {
    checkPermission(caller, permission, 'register.view')
  }
}
