/**
 * What the form of a connection is built from: the fields its type declares. A field with a fixed
 * value is held by the type and is not shown for editing.
 */
import type {
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  ConnectionValue,
} from './types'

/** The fields the person edits, in the order the type declares them. */
export function getEditableFields(type: ConnectionTypeInfo): readonly ConnectionFieldInfo[] {
  return type.fields.filter((one) => one.fixedValue === undefined)
}

/** The default of each plain field that declares one. A secret has none. */
export function createFieldValues(type: ConnectionTypeInfo): Record<string, ConnectionValue> {
  const values: Record<string, ConnectionValue> = {}
  for (const one of getEditableFields(type)) {
    if (one.type !== 'secret' && one.defaultValue !== undefined) values[one.key] = one.defaultValue
  }
  return values
}

/** The address field the person fills in, and none where the type fixes its address or has none. */
export function getAddressField(type: ConnectionTypeInfo): ConnectionFieldInfo | undefined {
  return getEditableFields(type).find((one) => one.isAddress)
}

/** The keys of the required editable fields that hold no value; a stored secret counts as one. */
export function getMissingFields(
  type: ConnectionTypeInfo,
  values: Readonly<Record<string, ConnectionValue>>,
  secrets: Readonly<Record<string, string>>,
  storedSecretKeys: readonly string[] = [],
): readonly string[] {
  return getEditableFields(type)
    .filter((one) => one.isRequired && !hasValue(one, values, secrets, storedSecretKeys))
    .map((one) => one.key)
}

function hasValue(
  field: ConnectionFieldInfo,
  values: Readonly<Record<string, ConnectionValue>>,
  secrets: Readonly<Record<string, string>>,
  storedSecretKeys: readonly string[],
): boolean {
  if (field.type === 'secret') {
    return (secrets[field.key] ?? '') !== '' || storedSecretKeys.includes(field.key)
  }
  const value = values[field.key]
  return typeof value === 'string' ? value.trim() !== '' : value !== undefined
}

function isAddressChanged(
  type: ConnectionTypeInfo,
  stored: Readonly<Record<string, ConnectionValue>>,
  typed: Readonly<Record<string, ConnectionValue>>,
): boolean {
  const address = getAddressField(type)
  return address !== undefined && (stored[address.key] ?? '') !== (typed[address.key] ?? '')
}

/**
 * The values the form opens with: the defaults of the first available type for a new connection,
 * and the stored values over the defaults of its type for an existing one.
 */
export function createInitialValues(
  types: readonly ConnectionTypeInfo[],
  connection?: ConnectionInfo,
): Record<string, ConnectionValue> {
  if (!connection) {
    const first = types.find((one) => one.isAvailable)
    return first ? createFieldValues(first) : {}
  }
  const own = types.find((one) => one.id === connection.typeId)
  const values: Record<string, ConnectionValue> = own ? createFieldValues(own) : {}
  return Object.assign(values, connection.fields)
}

/** The secret fields of a connection that still hold a key: none once the address was changed. */
export function getStoredSecretKeys(
  type: ConnectionTypeInfo,
  connection: ConnectionInfo | undefined,
  typed: Readonly<Record<string, ConnectionValue>>,
): readonly string[] {
  if (!connection) return []
  return isAddressChanged(type, connection.fields, typed) ? [] : connection.secretKeysSet
}

/** Whether saving removes a key the connection holds, because the address was changed. */
export function isStoredKeyLost(
  type: ConnectionTypeInfo,
  connection: ConnectionInfo | undefined,
  typed: Readonly<Record<string, ConnectionValue>>,
): boolean {
  return (
    connection !== undefined &&
    connection.secretKeysSet.length > 0 &&
    isAddressChanged(type, connection.fields, typed)
  )
}

/** The values of the editable fields that hold something; a fixed value is never sent. */
export function getSentFields(
  type: ConnectionTypeInfo,
  values: Readonly<Record<string, ConnectionValue>>,
): Record<string, ConnectionValue> {
  const editable = new Set(getEditableFields(type).map((one) => one.key))
  return Object.fromEntries(
    Object.entries(values).filter(([key, value]) => editable.has(key) && value !== ''),
  )
}

/** The keys typed, and an empty text for each key the person removed. */
export function getSentSecrets(
  secrets: Readonly<Record<string, string>>,
  clearedKeys: readonly string[],
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(secrets).filter(([key, value]) => value !== '' || clearedKeys.includes(key)),
  )
}

/** The lines under a field, by state. */
export interface FieldHints {
  readonly cleared: string
  readonly stored: string
  readonly requiredKey: string
  readonly optionalKey: string
  readonly address: string
}

/** The line shown under a field: the state of a secret, the use of an address, or the description. */
export function getFieldHint(
  field: ConnectionFieldInfo,
  state: { readonly isStored: boolean; readonly isCleared: boolean },
  hints: FieldHints,
): string {
  if (field.type !== 'secret') return field.isAddress ? hints.address : field.description
  if (state.isCleared) return hints.cleared
  if (state.isStored) return hints.stored
  return field.isRequired ? hints.requiredKey : hints.optionalKey
}
