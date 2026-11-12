/** A value that survives JSON.stringify and JSON.parse unchanged: the shape of an RPC body and of a setting. */
export type JsonValue = string | number | boolean | null | JsonValue[] | JsonObject

/** A JsonValue that is an object. */
export interface JsonObject {
  [key: string]: JsonValue
}
