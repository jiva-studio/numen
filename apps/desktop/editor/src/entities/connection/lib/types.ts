/**
 * What a connection is, as the window holds it: the kinds of connection the plugins provide, the
 * connections the person made of them, and the models a connection lists. No value of a secret is
 * ever read back.
 */

/** A value a field of a connection holds. */
export type ConnectionValue = string | number | boolean

/** The type of a field of a kind of connection. */
type ConnectionFieldType = 'string' | 'number' | 'boolean' | 'enum' | 'file' | 'secret'

/** One field of a kind of connection, as its form is built from it. */
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
}

/** A kind of connection one plugin provides. */
export interface ConnectionTypeInfo {
  readonly id: string
  readonly pluginId: string
  readonly pluginName: string
  readonly title: string
  readonly description: string
  readonly operations: readonly string[]
  readonly fields: readonly ConnectionFieldInfo[]
  readonly isAvailable: boolean
}

/** A setting of rows of a plugin that names a connection. */
interface ConnectionUse {
  readonly pluginId: string
  readonly pluginName: string
  readonly operation: string
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
  readonly isPrivateNetworkAllowed: boolean
  readonly uses: readonly ConnectionUse[]
}

/** A connection to create. An empty name asks the core to name it from the title of the type. */
export interface ConnectionDraft {
  readonly typeId: string
  readonly name: string
  readonly fields: Readonly<Record<string, ConnectionValue>>
  readonly secrets: Readonly<Record<string, string>>
  readonly isPrivateNetworkAllowed: boolean
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
type ConnectionErrorCode =
  | 'not_found'
  | 'name_taken'
  | 'invalid_field'
  | 'type_unavailable'
  | 'key_store_failed'
  | 'stale_revision'

/** A served refusal with the reason the core gives. */
export interface ConnectionFailure {
  readonly errorCode: ConnectionErrorCode
  readonly detail: string
}

/** A model a connection lists. `operations` is empty where the service labels no operation. */
export interface ModelInfo {
  readonly id: string
  readonly title: string
  readonly operations: readonly string[]
}

/** A reasoning level of a model. */
export interface ReasoningLevelInfo {
  readonly value: string
  readonly title: string
}

/** Why the core did not give a list. */
type ModelListErrorCode = 'not_found' | 'provider_unavailable'

/** A served refusal with the reason the core gives. */
export interface ModelListFailure {
  readonly errorCode: ModelListErrorCode
  readonly detail: string
}
