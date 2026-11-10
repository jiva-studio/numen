// JsonValue is a value that survives JSON.stringify and JSON.parse unchanged: the shape of a setting,
// of the state a tab keeps across a restart, and of a message to a backend.
export type JsonValue =
  | string
  | number
  | boolean
  | null
  | readonly JsonValue[]
  | { readonly [key: string]: JsonValue }

// JsonObject is a JsonValue that is an object.
export type JsonObject = { readonly [key: string]: JsonValue }
