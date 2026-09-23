/** The shell, which mounts every screen and is itself none. */
import { held } from '../note-tab/tab'
import { deck } from '../cards-tab/deck'

export const mounts = () => `${held()} ${deck()}`
