/**
 * The page of secrets and the menu of a secret field, as plain data: the sections of the secrets
 * by group, the plugin declarations that may be allowed a secret, and the order of the menu.
 */
import type {
  PluginInfo,
  SecretDeclaration,
  Secret,
  SecretMenu,
  SecretSection,
} from '@/entities/settings'
import { WORDS } from '../words'

export type { SecretSection }

/** One secret a plugin declares, and so one permission a secret can grant. */
export interface Declaration {
  readonly pluginId: string
  readonly pluginName: string
  readonly name: string
  readonly group: string
  readonly title: string
  readonly description: string
}

export function getDeclaration(plugin: PluginInfo, secret: SecretDeclaration): Declaration {
  return {
    pluginId: plugin.id,
    pluginName: plugin.name,
    name: secret.name,
    group: secret.group,
    title: secret.title,
    description: secret.description,
  }
}

/** Every secret the installed plugins declare. */
export function getDeclarations(plugins: readonly PluginInfo[]): readonly Declaration[] {
  return plugins.flatMap((plugin) => plugin.secrets.map((one) => getDeclaration(plugin, one)))
}

/** Builds the secret sections for the page, appending empty local groups and value states. */
export function createPageSections(
  sections: readonly SecretSection[],
  localGroups: readonly string[],
  states: ReadonlyMap<string, boolean>,
): readonly SecretSection[] {
  const coreGroups = new Set(sections.map((one) => one.group))
  const added = [...new Set(localGroups)]
    .filter((group) => !coreGroups.has(group))
    .map((group) => ({ group, isAsked: false, secrets: [] }))
  return [...sections, ...added].map((one) => ({
    ...one,
    secrets: one.secrets.map((secret) => ({ ...secret, hasValue: states.get(secret.id) ?? false })),
  }))
}

/**
 * The declarations a section offers, each plugin once for a name: those of the group, and for a
 * name the plugin declares no secret of this group for, the one with no group.
 */
export function getSectionDeclarations(
  declarations: readonly Declaration[],
  group: string,
): readonly Declaration[] {
  const ownKeys = new Set(
    declarations.filter((one) => one.group === group).map((one) => `${one.pluginId}/${one.name}`),
  )
  return declarations.filter(
    (one) =>
      one.group === group || (one.group === '' && !ownKeys.has(`${one.pluginId}/${one.name}`)),
  )
}

function getReaderNames(secret: Secret, plugins: readonly PluginInfo[]): readonly string[] {
  const ids = [...new Set(secret.allowed.map((one) => one.pluginId))]
  return ids.map((id) => plugins.find((one) => one.id === id)?.name ?? id)
}

/**
 * The secrets a declaration offers in its menu, in the order the core gave them.
 */
export function getMenuSecrets(
  menus: readonly SecretMenu[],
  declaration: Declaration,
): readonly Secret[] {
  const menu = menus.find(
    (one) =>
      one.pluginId === declaration.pluginId &&
      one.name === declaration.name &&
      one.group === declaration.group,
  )
  return menu?.secrets ?? []
}

/** Whether the group of the declaration holds a secret. */
export function hasGroupSecrets(
  sections: readonly SecretSection[],
  declaration: Declaration,
): boolean {
  return sections.some((one) => one.group === declaration.group && one.secrets.length > 0)
}

/** A secret or secret group selected for deletion. */
export type DeletionTarget =
  | { readonly kind: 'secret'; readonly secret: Secret }
  | { readonly kind: 'group'; readonly section: SecretSection }

/** The ids of the secrets a deletion removes. */
export function getDeletionIds(target: DeletionTarget): readonly string[] {
  return target.kind === 'secret' ? [target.secret.id] : target.section.secrets.map((one) => one.id)
}

/** Returns the confirmation message for deleting a secret or group. */
export function getDeletionText(target: DeletionTarget, plugins: readonly PluginInfo[]): string {
  if (target.kind === 'secret') {
    return WORDS.secretDeleteAsking(target.secret.name, getReaderNames(target.secret, plugins))
  }
  const { group, secrets } = target.section
  const readers = new Set(secrets.flatMap((one) => getReaderNames(one, plugins)))
  return WORDS.secretGroupDeleteAsking(group, secrets.length, [...readers])
}
