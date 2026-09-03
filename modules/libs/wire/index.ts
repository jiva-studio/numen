/**
 * What the windows' ports are answered with over the wire.
 *
 * `libs/ui` declares the ports and never sees the protocol; a window draws and
 * never translates. What turns one into the other stands here, once.
 */
export { agentPort } from './agent'
export type { Asking } from './agent'
