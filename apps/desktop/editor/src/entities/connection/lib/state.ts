import type { ConnectionInfo, ConnectionTypeInfo } from './types'
import { getAddressField } from './form'

/** Whether a connection can be used: it is unavailable, incomplete, or ready. */
export type ConnectionState = 'unavailable' | 'incomplete' | 'ready'

/** A connection whose type no plugin provides is unavailable whatever else it lacks. */
export function getConnectionState(connection: ConnectionInfo): ConnectionState {
  if (!connection.isTypeAvailable) return 'unavailable'
  return connection.isComplete ? 'ready' : 'incomplete'
}

const OPERATION_TITLES: Readonly<Record<string, string>> = {
  'llm.complete': 'Completion',
  'llm.embed': 'Embeddings',
}

function getOperationTitle(operation: string): string {
  const name = operation.slice(operation.lastIndexOf('.') + 1)
  return OPERATION_TITLES[operation] ?? name.charAt(0).toUpperCase() + name.slice(1)
}

/**
 * The titles of what a type can do, one for each operation; none where the type is not known. An
 * operation that only lists the options of a row field is no title.
 */
export function getOperationTitles(type: ConnectionTypeInfo | undefined): readonly string[] {
  const lists = new Set((type?.rowFields ?? []).map((one) => one.optionsFrom))
  return (type?.operations ?? []).filter((one) => !lists.has(one)).map(getOperationTitle)
}

/** The titles of what a type can do, joined; empty where the type is not known. */
export function getOperationsText(type: ConnectionTypeInfo | undefined): string {
  return getOperationTitles(type).join(' · ')
}

/** The address the person entered for a connection; empty for a fixed address or an unknown type. */
export function getAddressText(
  type: ConnectionTypeInfo | undefined,
  connection: ConnectionInfo,
): string {
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
