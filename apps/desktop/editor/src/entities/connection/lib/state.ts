import type { ConnectionInfo, ConnectionTypeInfo } from './types'
import { getAddressField } from './form'

/** Whether a connection can be used: it is unavailable, incomplete, or ready. */
export type ConnectionState = 'unavailable' | 'incomplete' | 'ready'

/** A connection whose type no plugin provides is unavailable whatever else it lacks. */
export function getConnectionState(connection: ConnectionInfo): ConnectionState {
  if (!connection.isTypeAvailable) return 'unavailable'
  return connection.isComplete ? 'ready' : 'incomplete'
}

/**
 * Where a connection points: for a detected connection the path its provider found it at, otherwise the
 * address the person entered; empty for a fixed address or an unknown type.
 */
export function getLocationText(
  type: ConnectionTypeInfo | undefined,
  connection: ConnectionInfo,
): string {
  if (connection.isDetected) return connection.path
  const field = type ? getAddressField(type) : undefined
  const value = field ? connection.fields[field.key] : undefined
  return typeof value === 'string' ? value : ''
}

/** Whether an installed plugin provides at least one type. */
export function hasAvailableType(types: readonly ConnectionTypeInfo[]): boolean {
  return types.some((one) => one.isAvailable)
}

/** Whether some connection has a type no installed plugin provides. */
export function hasUnavailableConnection(connections: readonly ConnectionInfo[]): boolean {
  return connections.some((one) => !one.isTypeAvailable)
}

/** The names of the plugins whose rows use a connection, once each, as a list a person reads. */
export function getUsedByNames(uses: ConnectionInfo['uses']): string {
  const names = [...new Set(uses.map((one) => one.pluginName))]
  const last = names.pop()
  if (last === undefined) return ''
  return names.length === 0 ? last : `${names.join(', ')} and ${last}`
}
