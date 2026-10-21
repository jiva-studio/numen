/** The reader tab kinds of a window. */
import type { WindowKindsDeps } from './deps'

export type ReaderKindsDeps = Pick<WindowKindsDeps, 'tabOpeners' | 'windowTabs'>

export function createReaderKinds(_deps?: ReaderKindsDeps) {
  return {}
}
