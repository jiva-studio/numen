/**
 * A folder the rule holds no part. It is a screen because it is neither
 * `shared/` nor `window/`, which is how a folder added tomorrow is covered
 * without being listed.
 */
import { held } from '../note-tab/tab'

export const entries = () => held()
