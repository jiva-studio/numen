/**
 * The rules for the names of secrets and groups: how a name is trimmed, how long it may be, the
 * names a new secret or group is offered, and the order moves of a secret.
 */
import { findServiceInfo } from '@numen/core-client'
import type { Secret } from './configuration'

/** The longest name of a secret or group, in characters; the core refuses a longer one. */
export const MAX_NAME_LENGTH = 64

const DEFAULT_SECRET_NAME = 'Default'
const NEW_GROUP_NAME = 'New group'

/** The characters Go's `unicode.IsSpace` accepts, which `strings.TrimSpace` removes at the ends. */
const SPACE =
  '\\t\\n\\v\\f\\r \\u0085\\u00A0\\u1680\\u2000-\\u200A\\u2028\\u2029\\u202F\\u205F\\u3000'
const ENDS = new RegExp(`^[${SPACE}]+|[${SPACE}]+$`, 'gu')

/** The name without the space at its ends, as the core trims the name of a secret or group. */
export function trimName(name: string): string {
  return name.replace(ENDS, '')
}

/** Whether the name is longer than the core accepts. */
export function isNameTooLong(name: string): boolean {
  return [...name].length > MAX_NAME_LENGTH
}

/** Whether renaming a group to the name would collide with another group of the page. */
export function isGroupNameTaken(groups: readonly string[], group: string, name: string): boolean {
  return name !== group && groups.includes(name)
}

/** The name of a new group: New group, else the first number after it that no group has. */
export function getNewGroupName(groups: readonly string[]): string {
  if (!groups.includes(NEW_GROUP_NAME)) return NEW_GROUP_NAME
  let number = 2
  while (groups.includes(`${NEW_GROUP_NAME} ${number}`)) number++
  return `${NEW_GROUP_NAME} ${number}`
}

/**
 * The name the first secret of a group gets: the name of its service, or Default for a group no
 * service is known by.
 */
export function getFirstSecretName(group: string): string {
  const title = findServiceInfo(group)?.title ?? ''
  return title === '' ? DEFAULT_SECRET_NAME : title
}

/**
 * The name a new secret of a group gets: the name of the service in a group with no secret, else
 * the name with V1, V2 and on, the first one not in use.
 */
export function getNextSecretName(group: string, namesInGroup: readonly string[]): string {
  const first = getFirstSecretName(group)
  if (namesInGroup.length === 0) return first
  let number = 1
  while (namesInGroup.includes(`${first} V${number}`)) number++
  return `${first} V${number}`
}

/** What a secret grants to a plugin: the plugin, the name it declares and the group it asks for. */
export interface PermissionKey {
  readonly pluginId: string
  readonly name: string
  readonly group: string
}

/** Whether the permission is granted on the secret. */
export function isAllowed(secret: Secret, key: PermissionKey): boolean {
  return secret.allowed.some(
    (one) => one.pluginId === key.pluginId && one.name === key.name && one.group === key.group,
  )
}

/** The place a secret is moved to: the secret, and the one it goes before, or none for the end. */
export interface MoveTarget {
  readonly id: string
  readonly beforeId: string
}

/**
 * Where the secret at the index goes when it moves one place up or down, and nothing where it is
 * already first or last.
 */
export function getMoveTarget(
  secrets: readonly { readonly id: string }[],
  index: number,
  direction: 'up' | 'down',
): MoveTarget | null {
  const secret = secrets[index]
  if (!secret) return null
  if (direction === 'up') {
    const above = secrets[index - 1]
    return above ? { id: secret.id, beforeId: above.id } : null
  }
  if (index === secrets.length - 1) return null
  return { id: secret.id, beforeId: secrets[index + 2]?.id ?? '' }
}
