/**
 * The names a new secret of a group is offered: the title of the service the group is named after,
 * and a numbered variant of it where the group already has secrets.
 */
import { findServiceInfo } from '../api/services'

const DEFAULT_SECRET_NAME = 'Default'

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
