/**
 * Where a node's identity comes from.
 *
 * It is used in the DOM as the id of a splitter panel, so it is unique to the
 * document and not only to this tree.
 */
import type { NodeIdFactory } from './splice'

/** What a name is drawn from. */
export interface Randomness {
  readonly uuid: () => string | null
}

/**
 * Identities, counted from one where the machine makes none of its own. The
 * count belongs to the factory, so two of them never meet.
 */
export function createNodeIdFactory(random: Randomness): NodeIdFactory {
  let createdItem = 0
  return () => random.uuid() ?? `node-${++createdItem}`
}
