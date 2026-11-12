import type { JsonObject, JsonValue } from '@/shared/api/json'
import type { ConnectionInfo, ConnectionTypeInfo, ModelInfo } from './types'

/** A row as the core stores it: a connection, a model and a reasoning level; empty is the default. */
export interface StoredRow {
  readonly connection: string
  readonly model: string
  readonly reasoning: string
}

/** A row on the page, with a key that stays while it is moved. It may not be complete yet. */
export interface RowDraft extends StoredRow {
  readonly key: string
}

/** The words a connection that cannot be used is described in. */
export interface UsabilityWords {
  readonly unavailable: string
  readonly incomplete: string
  readonly cannot: (operation: string) => string
}

/** Whether a connection can serve an operation, and if not, why. */
export interface Usability {
  readonly isUsable: boolean
  readonly reason: string
}

const readText = (value: JsonValue | undefined): string | undefined =>
  typeof value === 'string' ? value : undefined

const isRecord = (value: JsonValue | undefined): value is JsonObject =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

/** The rows in the value of a setting of rows; a value that is no list of rows holds none. */
export function readRows(value: JsonValue | undefined): readonly StoredRow[] {
  if (!Array.isArray(value)) return []
  const rows: StoredRow[] = []
  for (const one of value) {
    if (!isRecord(one)) return []
    const connection = readText(one['connection'])
    const model = readText(one['model'])
    if (connection === undefined || model === undefined) return []
    rows.push({ connection, model, reasoning: readText(one['reasoning']) ?? '' })
  }
  return rows
}

/** The rows that name a connection and a model: the only ones the core stores. */
export function getCompleteRows(drafts: readonly RowDraft[]): readonly StoredRow[] {
  return drafts
    .filter((one) => one.connection !== '' && one.model !== '')
    .map((one) => ({ connection: one.connection, model: one.model, reasoning: one.reasoning }))
}

/** The rows with the one at `from` put at `to`. A place outside the list leaves them as they were. */
export function moveRow<T>(rows: readonly T[], from: number, to: number): readonly T[] {
  const isInside = (index: number): boolean => index >= 0 && index < rows.length
  if (from === to || !isInside(from) || !isInside(to)) return rows
  const moved = [...rows]
  moved.splice(to, 0, ...moved.splice(from, 1))
  return moved
}

/** Whether a connection can serve the operation: its type is installed, complete and does it. */
export function getConnectionUsability(
  connection: ConnectionInfo,
  type: ConnectionTypeInfo | undefined,
  operation: string,
  words: UsabilityWords,
): Usability {
  if (!connection.isTypeAvailable || type === undefined || !type.isAvailable) {
    return { isUsable: false, reason: words.unavailable }
  }
  if (!type.operations.includes(operation)) {
    return { isUsable: false, reason: words.cannot(operation) }
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

/** The models a service labels for the operation, and those it labels for none. */
export function listModelsForOperation(
  models: readonly ModelInfo[],
  operation: string,
): readonly ModelInfo[] {
  return models.filter((one) => one.operations.length === 0 || one.operations.includes(operation))
}
