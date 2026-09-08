/** A feature reaching another feature, which is the one thing the rule refuses. */
import { turn } from '../thread/turn'
import { beside } from '../../shared/lib/place'

export const deck = () => `${turn()} ${beside(3)}`
