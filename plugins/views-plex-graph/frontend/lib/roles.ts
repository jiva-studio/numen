/**
 * The roles a gesture of the plex can write.
 */
import type { ConnectionRole } from '@numen/plugin-sdk'
import type { PlexRelation } from '../plex'

/** The role a relation is written as, among the roles the host offers; undefined for a relation that is none of them. */
export function findRole(
  roles: readonly ConnectionRole[],
  relation: PlexRelation,
): ConnectionRole | undefined {
  return roles.find((role) => role === relation)
}
