/** A screen reaching another screen, which is the one thing the rule refuses. */
import { held } from '../note/tab'
import { puts } from '../tabs/putting'

export const deck = () => `${held()} ${puts()}`
