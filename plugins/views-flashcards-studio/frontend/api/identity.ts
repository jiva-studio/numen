import type { IdMaker } from '../lib/identity'

/** A random identity from the window. */
export const generateId: IdMaker = () => crypto.randomUUID()
