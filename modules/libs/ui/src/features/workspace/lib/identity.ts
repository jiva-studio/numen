/**
 * Where a node's identity comes from.
 *
 * It reaches the DOM as the id of a splitter panel, so it is unique to the
 * document and not only to this tree.
 */
import type { NodeId } from './node'
import type { NodeIdFactory } from './splice'

/** What a name is drawn from. The window's own randomness by default. */
export interface Randomness {
  /** An identifier no other call gives, where this machine can make one. */
  readonly uuid: () => string | null
}

export const browserRandomness: Randomness = {
  uuid: () => (typeof crypto?.randomUUID === 'function' ? crypto.randomUUID() : null),
}

/**
 * Identities, counted from one where the machine makes none of its own. The
 * count belongs to the factory, so two of them never meet.
 */
export function createNodeIdFactory(random: Randomness = browserRandomness): NodeIdFactory {
  let made = 0
  return () => (random.uuid() ?? `node-${++made}`) as NodeId
}
