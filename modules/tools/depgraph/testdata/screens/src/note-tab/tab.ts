/** A screen reaching `shared/` and the window's root: both permitted. */
import { puts } from '../shared/tabs/putting'
import { WORDS } from '../words'

export const held = () => `${puts()} ${WORDS.going}`
