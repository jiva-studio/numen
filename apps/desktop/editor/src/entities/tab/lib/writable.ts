/** The type with its readonly properties made writable, for an object built up before it is returned. */
export type Writable<T> = { -readonly [K in keyof T]: T[K] }
