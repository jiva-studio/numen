/**
 * A screen reaching a feature through its door and the shared layer, both of
 * which it may, and one file past that door, which it may not.
 */
import { turn } from '../features/thread'
import { editor } from '../features/cards/deck-editor/editor'
import { item } from '../shared/ui/menu/item'

export const agent = () => `${turn()} ${editor()} ${item()}`
