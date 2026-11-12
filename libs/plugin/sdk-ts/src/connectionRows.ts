import { asFailure, asValue, type Result } from './result'
import type { JsonObject, JsonValue } from './types'

/**
 * One entry of a setting of rows: the connection, the model and the reasoning level the person chose.
 * The connection is an opaque id; the fields of the connection never reach a plugin. An empty
 * reasoning asks for the default of the provider.
 */
export interface ConnectionRow {
  readonly connection: string
  readonly model: string
  readonly reasoning: string
}

const isText = (value: JsonValue | undefined): value is string => typeof value === 'string'

const isRecord = (value: JsonValue | undefined): value is JsonObject =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const readRow = (value: JsonValue | undefined): ConnectionRow | undefined => {
  if (!isRecord(value)) return undefined
  const connection = value['connection']
  const model = value['model']
  const reasoning = value['reasoning']
  if (!isText(connection) || !isText(model)) return undefined
  if (reasoning !== undefined && !isText(reasoning)) return undefined
  return { connection, model, reasoning: reasoning ?? '' }
}

/**
 * Reads the rows of a setting of type connections from the value `getSetting` returns, in the order
 * of use. A setting that was not delivered has no row.
 */
export const readConnectionRows = (
  value: JsonValue | undefined,
): Result<readonly ConnectionRow[], 'invalid_rows'> => {
  if (value === undefined) return asValue([])
  if (!Array.isArray(value)) return asFailure('invalid_rows')
  const list: readonly JsonValue[] = value
  const rows: ConnectionRow[] = []
  for (const one of list) {
    const row = readRow(one)
    if (row === undefined) return asFailure('invalid_rows')
    rows.push(row)
  }
  return asValue(rows)
}
