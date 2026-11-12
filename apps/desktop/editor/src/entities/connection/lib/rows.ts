import type { JsonObject, JsonValue } from '@/shared/api/json'
import type {
  ConnectionFieldInfo,
  ConnectionInfo,
  ConnectionTypeInfo,
  OperationTitles,
} from './types'

/**
 * A row as the core stores it: a connection and the values of the row fields of its type, such as a
 * model. A field with no value is left out.
 */
export interface StoredRow {
  readonly connection: string
  readonly options: Readonly<Record<string, string>>
}

/** A row on the page, with a key that stays while it is moved. It may not be complete yet. */
export interface RowDraft extends StoredRow {
  readonly key: string
}

/** The words a connection that cannot be used is described in. */
export interface UsabilityWords {
  readonly unavailable: string
  readonly incomplete: string
  // Takes the title of the operation, or its name where no plugin titles it.
  readonly cannot: (operation: string) => string
}

/** Whether a connection can serve the operations of a setting, and if not, why. */
export interface Usability {
  readonly isUsable: boolean
  readonly reason: string
}

/** The key a row is stored under for its connection; every other key is a row field. */
const CONNECTION_KEY = 'connection'

const isRecord = (value: JsonValue | undefined): value is JsonObject =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const readRow = (value: JsonValue | undefined): StoredRow | undefined => {
  if (!isRecord(value)) return undefined
  const connection = value[CONNECTION_KEY]
  if (typeof connection !== 'string') return undefined
  const options: Record<string, string> = {}
  for (const [key, one] of Object.entries(value)) {
    if (key === CONNECTION_KEY) continue
    if (typeof one !== 'string') return undefined
    if (one !== '') options[key] = one
  }
  return { connection, options }
}

/** The rows in the value of a setting of rows; a value that is no list of rows holds none. */
export function readRows(value: JsonValue | undefined): readonly StoredRow[] {
  if (!Array.isArray(value)) return []
  const rows: StoredRow[] = []
  for (const one of value) {
    const row = readRow(one)
    if (row === undefined) return []
    rows.push(row)
  }
  return rows
}

/** The rows as the value of a setting of rows: the connection and each row field under its key. */
export function writeRows(rows: readonly StoredRow[]): JsonObject[] {
  return rows.map((one) => ({ [CONNECTION_KEY]: one.connection, ...one.options }))
}

/** The text a row is compared by: its connection and its values in the order of their keys. */
export function getRowKey(row: StoredRow): string {
  const values = Object.keys(row.options)
    .sort()
    .map((key) => `${key}=${row.options[key] ?? ''}`)
  return [row.connection, ...values].join('\0')
}

/**
 * The rows that name a connection and hold a value for each required row field of its type: the only
 * ones the core stores.
 */
export function getCompleteRows(
  drafts: readonly RowDraft[],
  getRowFields: (connectionId: string) => readonly ConnectionFieldInfo[],
): readonly StoredRow[] {
  return drafts
    .filter(
      (one) =>
        one.connection !== '' &&
        getRowFields(one.connection).every(
          (field) => !field.isRequired || (one.options[field.key] ?? '') !== '',
        ),
    )
    .map((one) => ({
      connection: one.connection,
      options: Object.fromEntries(Object.entries(one.options).filter(([, value]) => value !== '')),
    }))
}

/** The rows with the one at `from` put at `to`. A place outside the list leaves them as they were. */
export function moveRow<T>(rows: readonly T[], from: number, to: number): readonly T[] {
  const isInside = (index: number): boolean => index >= 0 && index < rows.length
  if (from === to || !isInside(from) || !isInside(to)) return rows
  const moved = [...rows]
  moved.splice(to, 0, ...moved.splice(from, 1))
  return moved
}

/** Whether a connection can serve the operations: its type is installed, complete and serves each. */
export function getConnectionUsability(
  connection: ConnectionInfo,
  type: ConnectionTypeInfo | undefined,
  operations: readonly string[],
  words: UsabilityWords,
  operationTitles: OperationTitles,
): Usability {
  if (!connection.isTypeAvailable || type === undefined || !type.isAvailable) {
    return { isUsable: false, reason: words.unavailable }
  }
  const missing = operations.find((one) => !type.operations.includes(one))
  if (missing !== undefined) {
    return { isUsable: false, reason: words.cannot(operationTitles[missing] ?? missing) }
  }
  if (!connection.isComplete) return { isUsable: false, reason: words.incomplete }
  return { isUsable: true, reason: '' }
}

/** The usable items first and the unusable after them, each group in the order it came. */
export function sortByUsability<T extends { readonly isUsable: boolean }>(
  items: readonly T[],
): readonly T[] {
  return [...items.filter((one) => one.isUsable), ...items.filter((one) => !one.isUsable)]
}
