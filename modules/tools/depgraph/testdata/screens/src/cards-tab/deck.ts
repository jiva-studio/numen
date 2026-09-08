/** A screen reaching another screen, which is the one thing the rule refuses. */
import { held } from '../note-tab/tab'
import { core } from '../shared/core'

export const deck = () => `${held()} ${core()}`
