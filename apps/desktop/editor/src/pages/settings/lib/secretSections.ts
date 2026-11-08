/**
 * The page of secrets and the menu of a secret field, as plain data: the sections of the secrets
 * by group, the plugin declarations that may be allowed a secret, and the order of the menu.
 */
import type { PluginInfo, PluginSecret, Secret } from '@/entities/settings'
import { WORDS } from '../words'

/** The secrets of one group in priority order. */
export interface SecretSection {
  readonly group: string
  readonly isAsked: boolean
  readonly secrets: readonly Secret[]
}

/** One secret a plugin declares, and so one permission a secret can grant. */
export interface Declaration {
  readonly pluginId: string
  readonly pluginName: string
  readonly name: string
  readonly group: string
  readonly title: string
  readonly description: string
}

const encoder = new TextEncoder()

function compareNames(first: string, second: string): number {
  const one = encoder.encode(first)
  const other = encoder.encode(second)
  const shared = Math.min(one.length, other.length)
  for (let index = 0; index < shared; index++) {
    const difference = (one[index] ?? 0) - (other[index] ?? 0)
    if (difference !== 0) return difference
  }
  return one.length - other.length
}

/** The declaration of one secret of a plugin. */
export function getDeclaration(plugin: PluginInfo, secret: PluginSecret): Declaration {
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

/**
 * The sections of the page: the groups the installed plugins ask for by name, then the groups the
 * person made by name. A group the plugins ask for is a section while it has no secret. The
 * extra groups are groups the person made that hold no secret yet.
 */
export function createSecretSections(
  plugins: readonly PluginInfo[],
  secrets: readonly Secret[],
  extraGroups: readonly string[] = [],
): readonly SecretSection[] {
  const pluginGroups = new Set(
    getDeclarations(plugins)
      .map((one) => one.group)
      .filter((one) => one !== ''),
  )
  const made = new Set(
    [...secrets.map((one) => one.group), ...extraGroups].filter((one) => !pluginGroups.has(one)),
  )
  const sorted = (groups: ReadonlySet<string>): string[] => [...groups].sort(compareNames)
  return [
    ...sorted(pluginGroups).map((group) => ({ group, isAsked: true })),
    ...sorted(made).map((group) => ({ group, isAsked: false })),
  ].map((one) => ({ ...one, secrets: secrets.filter((secret) => secret.group === one.group) }))
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

/** The names of the plugins that may read the secret, each once, in the order of the permissions. */
export function getReaderNames(secret: Secret, plugins: readonly PluginInfo[]): readonly string[] {
  const ids = [...new Set(secret.allowed.map((one) => one.pluginId))]
  return ids.map((id) => plugins.find((one) => one.id === id)?.name ?? id)
}

/**
 * The secrets a declaration offers in its menu: those of the declared group first, then the others
 * in the order of the page. A declaration with no group has no group of its own.
 */
export function getMenuSecrets(
  sections: readonly SecretSection[],
  declaration: Declaration,
): readonly Secret[] {
  const own = sections.filter((one) => declaration.group !== '' && one.group === declaration.group)
  const others = sections.filter((one) => !own.includes(one))
  return [...own, ...others].flatMap((one) => one.secrets)
}

/** Whether the group of the declaration holds a secret. */
export function hasGroupSecrets(
  sections: readonly SecretSection[],
  declaration: Declaration,
): boolean {
  return sections.some((one) => one.group === declaration.group && one.secrets.length > 0)
}

/** What the person asked to delete: one secret, or every secret of a group. */
export type DeletionTarget =
  | { readonly kind: 'secret'; readonly secret: Secret }
  | { readonly kind: 'group'; readonly section: SecretSection }

/** The ids of the secrets a deletion removes. */
export function getDeletionIds(target: DeletionTarget): readonly string[] {
  return target.kind === 'secret' ? [target.secret.id] : target.section.secrets.map((one) => one.id)
}

/** The question the confirmation asks, naming the plugins that lose a secret. */
export function getDeletionText(target: DeletionTarget, plugins: readonly PluginInfo[]): string {
  if (target.kind === 'secret') {
    return WORDS.secretDeleteAsking(target.secret.name, getReaderNames(target.secret, plugins))
  }
  const { group, secrets } = target.section
  const readers = new Set(secrets.flatMap((one) => getReaderNames(one, plugins)))
  return WORDS.secretGroupDeleteAsking(group, secrets.length, [...readers])
}
