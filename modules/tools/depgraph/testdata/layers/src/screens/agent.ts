/** A screen reaching a feature and the shared layer, both of which it may. */
import { turn } from '../features/thread/turn'
import { item } from '../shared/ui/menu/item'

export const agent = () => `${turn()} ${item()}`
