/**
 * The connections to services and programs: the kinds of connection the plugins provide, and the
 * connections the person made of them. No value of a secret is ever read back.
 */
import { createClient, type Client } from '@connectrpc/connect'
import { fromJson, toJson, type JsonValue } from '@bufbuild/protobuf'
import { ValueSchema, type Value } from '@bufbuild/protobuf/wkt'
import {
  ProviderConnectionErrorCode,
  ProviderConnectionService,
  type ManifestConnectionField,
  type ProviderConnection,
  type ProviderConnectionType,
} from '@numen/protocol'
import { asFailure, asValue, type Result } from '../result'
import { transport } from '../transport'

/** A value a field of a connection holds. */
export type ConnectionValue = string | number | boolean

/** The type of a field of a kind of connection. */
export type ConnectionFieldType = 'string' | 'number' | 'boolean' | 'enum' | 'file' | 'secret'

/**
 * One field of a kind of connection, as its form is built from it. optionsFrom is the operation of the type
 * that lists the values of a row field, empty for none.
 */
export interface ConnectionFieldInfo {
  readonly key: string
  readonly type: ConnectionFieldType
  readonly title: string
  readonly description: string
  readonly defaultValue?: ConnectionValue
  readonly fixedValue?: string
  readonly isRequired: boolean
  readonly isAddress: boolean
  readonly options: readonly { readonly value: string; readonly title: string }[]
  readonly min?: number
  readonly max?: number
  readonly step?: number
  readonly optionsFrom?: string
}

/**
 * A kind of connection one plugin provides. rowFields are the values a row that names a connection of the
 * type holds besides the connection.
 */
export interface ConnectionTypeInfo {
  readonly id: string
  readonly pluginId: string
  readonly pluginName: string
  readonly title: string
  readonly description: string
  readonly operations: readonly string[]
  readonly fields: readonly ConnectionFieldInfo[]
  readonly rowFields: readonly ConnectionFieldInfo[]
  readonly isAvailable: boolean
}

/** An operation a plugin titles: `title` completes "This type cannot ...". */
export interface ConnectionOperationInfo {
  readonly name: string
  readonly title: string
}

/** The kinds of connection the plugins provide, and the titles the plugins give operations. */
export interface ConnectionTypeList {
  readonly types: readonly ConnectionTypeInfo[]
  readonly operations: readonly ConnectionOperationInfo[]
}

/** A setting of rows of a plugin that names a connection. */
export interface ConnectionUse {
  readonly pluginId: string
  readonly pluginName: string
  readonly operations: readonly string[]
}

/** A stored connection. */
export interface ConnectionInfo {
  readonly id: string
  readonly typeId: string
  readonly name: string
  readonly revision: number
  readonly fields: Readonly<Record<string, ConnectionValue>>
  readonly secretKeysSet: readonly string[]
  readonly isTypeAvailable: boolean
  readonly isComplete: boolean
  readonly uses: readonly ConnectionUse[]
}

/** A connection to create. An empty name asks the core to name it from the title of the type. */
export interface ConnectionDraft {
  readonly typeId: string
  readonly name: string
  readonly fields: Readonly<Record<string, ConnectionValue>>
  readonly secrets: Readonly<Record<string, string>>
}

/**
 * A change of a connection. A field left out of `fields` is cleared. A secret left out of `secrets`
 * keeps its value, and an empty text clears it.
 */
export interface ConnectionChange extends Omit<ConnectionDraft, 'typeId'> {
  readonly id: string
  readonly revision: number
}

/** Why the core did not do what a call asked. */
export type ConnectionErrorCode =
  | 'not_found'
  | 'name_taken'
  | 'invalid_field'
  | 'type_unavailable'
  | 'key_store_failed'
  | 'stale_revision'
  | 'provider_unavailable'

/** A served refusal with the reason the core gives. */
export interface ConnectionFailure {
  readonly errorCode: ConnectionErrorCode
  readonly detail: string
}

/** The calls of ProviderConnectionService the connections are managed through. */
export type ConnectionRpc = Pick<
  Client<typeof ProviderConnectionService>,
  | 'listProviderConnectionTypes'
  | 'listProviderConnections'
  | 'createProviderConnection'
  | 'updateProviderConnection'
  | 'deleteProviderConnection'
  | 'listRowOptions'
>

/** The connections of the application. A call that cannot reach the core throws. */
export interface ConnectionClient {
  listTypes(): Promise<ConnectionTypeList>
  list(): Promise<readonly ConnectionInfo[]>
  create(draft: ConnectionDraft): Promise<Result<ConnectionInfo, ConnectionFailure>>
  update(change: ConnectionChange): Promise<Result<ConnectionInfo, ConnectionFailure>>
  delete(id: string): Promise<Result<undefined, ConnectionFailure>>
  /**
   * The values a row field of the type of a connection can take, as its provider lists them for the
   * values the row holds so far. `operations` are the operations the setting of rows is used for.
   */
  listRowOptions(
    connectionId: string,
    field: string,
    options: Readonly<Record<string, string>>,
    operations: readonly string[],
  ): Promise<Result<readonly RowOption[], ConnectionFailure>>
}

/** One value a row field can take. `title` is the value where the provider gives none. */
export interface RowOption {
  readonly value: string
  readonly title: string
  readonly group: string
}

const toWire = (value: ConnectionValue): Value => fromJson(ValueSchema, value)

const toWireFields = (fields: Readonly<Record<string, ConnectionValue>>): Record<string, Value> =>
  Object.fromEntries(Object.entries(fields).map(([key, value]) => [key, toWire(value)]))

const isConnectionValue = (value: JsonValue): value is ConnectionValue =>
  typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'

const readValue = (value: Value | undefined): ConnectionValue | undefined => {
  if (value === undefined) return undefined
  const json = toJson(ValueSchema, value)
  return isConnectionValue(json) ? json : undefined
}

const readOptions = (field: ManifestConnectionField): ConnectionFieldInfo['options'] =>
  (field.options?.values ?? []).flatMap((one) => {
    const json = toJson(ValueSchema, one)
    if (typeof json === 'string') return [{ value: json, title: json }]
    if (typeof json === 'object' && json !== null && !Array.isArray(json)) {
      const { value, title } = json
      if (typeof value !== 'string') return []
      return [{ value, title: typeof title === 'string' && title !== '' ? title : value }]
    }
    return []
  })

const readFieldType = (type: string): ConnectionFieldType => {
  const fieldTypes: readonly ConnectionFieldType[] = [
    'string',
    'number',
    'boolean',
    'enum',
    'file',
    'secret',
  ]
  return fieldTypes.find((one) => one === type) ?? 'string'
}

const readField = (field: ManifestConnectionField): ConnectionFieldInfo => {
  const defaultValue = readValue(field.default)
  return {
    key: field.key,
    type: readFieldType(field.type),
    title: field.title,
    description: field.description,
    ...(defaultValue === undefined ? {} : { defaultValue }),
    ...(field.fixedValue === undefined ? {} : { fixedValue: field.fixedValue }),
    isRequired: field.isRequired,
    isAddress: field.isAddress,
    options: readOptions(field),
    ...(field.min === undefined ? {} : { min: field.min }),
    ...(field.max === undefined ? {} : { max: field.max }),
    ...(field.step === undefined ? {} : { step: field.step }),
    ...(field.optionsFrom === '' ? {} : { optionsFrom: field.optionsFrom }),
  }
}

const readType = (type: ProviderConnectionType): ConnectionTypeInfo => ({
  id: type.typeId,
  pluginId: type.pluginId,
  pluginName: type.pluginName,
  title: type.title,
  description: type.description,
  operations: type.operations,
  fields: type.fields.map(readField),
  rowFields: type.rowFields.map(readField),
  isAvailable: type.isAvailable,
})

const readConnection = (connection: ProviderConnection): ConnectionInfo => {
  const fields: Record<string, ConnectionValue> = {}
  for (const [key, value] of Object.entries(connection.fields)) {
    const read = readValue(value)
    if (read !== undefined) fields[key] = read
  }
  return {
    id: connection.id,
    typeId: connection.typeId,
    name: connection.name,
    revision: connection.revision,
    fields,
    secretKeysSet: connection.secretKeysSet,
    isTypeAvailable: connection.isTypeAvailable,
    isComplete: connection.isComplete,
    uses: connection.uses.map((one) => ({
      pluginId: one.pluginId,
      pluginName: one.pluginName,
      operations: one.operations,
    })),
  }
}

const readErrorCode = (code: ProviderConnectionErrorCode): ConnectionErrorCode | undefined => {
  switch (code) {
    case ProviderConnectionErrorCode.NOT_FOUND:
      return 'not_found'
    case ProviderConnectionErrorCode.NAME_TAKEN:
      return 'name_taken'
    case ProviderConnectionErrorCode.INVALID_FIELD:
      return 'invalid_field'
    case ProviderConnectionErrorCode.TYPE_UNAVAILABLE:
      return 'type_unavailable'
    case ProviderConnectionErrorCode.KEY_STORE_FAILED:
      return 'key_store_failed'
    case ProviderConnectionErrorCode.STALE_REVISION:
      return 'stale_revision'
    case ProviderConnectionErrorCode.PROVIDER_UNAVAILABLE:
      return 'provider_unavailable'
    case ProviderConnectionErrorCode.UNSPECIFIED:
      return undefined
  }
}

const readFailure = (
  code: ProviderConnectionErrorCode,
  detail: string,
): ConnectionFailure | undefined => {
  const errorCode = readErrorCode(code)
  return errorCode === undefined ? undefined : { errorCode, detail }
}

const readAnswer = (answer: {
  error: ProviderConnectionErrorCode
  detail: string
  connection?: ProviderConnection | undefined
}): Result<ConnectionInfo, ConnectionFailure> => {
  const failure = readFailure(answer.error, answer.detail)
  if (failure) return asFailure(failure)
  if (answer.connection === undefined) {
    return asFailure({ errorCode: 'not_found', detail: 'The core returned no connection.' })
  }
  return asValue(readConnection(answer.connection))
}

/** The connections managed through a ProviderConnectionService client. */
export const createConnectionClient = (rpc: ConnectionRpc): ConnectionClient => ({
  listTypes: async () => {
    const answer = await rpc.listProviderConnectionTypes({})
    return {
      types: answer.types.map(readType),
      operations: answer.operations.map(({ name, title }) => ({ name, title })),
    }
  },
  list: async () => (await rpc.listProviderConnections({})).connections.map(readConnection),
  create: async (draft) =>
    readAnswer(
      await rpc.createProviderConnection({
        typeId: draft.typeId,
        name: draft.name,
        fields: toWireFields(draft.fields),
        secrets: { ...draft.secrets },
      }),
    ),
  update: async (change) =>
    readAnswer(
      await rpc.updateProviderConnection({
        id: change.id,
        revision: change.revision,
        name: change.name,
        fields: toWireFields(change.fields),
        secrets: { ...change.secrets },
      }),
    ),
  delete: async (id) => {
    const answer = await rpc.deleteProviderConnection({ id })
    const failure = readFailure(answer.error, answer.detail)
    return failure ? asFailure(failure) : asValue(undefined)
  },
  listRowOptions: async (connectionId, field, options, operations) => {
    const answer = await rpc.listRowOptions({
      connectionId,
      field,
      options: { ...options },
      operations: [...operations],
    })
    const failure = readFailure(answer.error, answer.detail)
    if (failure) return asFailure(failure)
    return asValue(
      answer.options.map((one) => ({
        value: one.value,
        title: one.title === '' ? one.value : one.title,
        group: one.group,
      })),
    )
  },
})

/** The connections of the window the page is served from. */
export const connections = createConnectionClient(
  createClient(ProviderConnectionService, transport),
)
