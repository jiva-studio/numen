/** Media kinds dependencies. */
import type { WindowKindsDeps } from './deps'

export type MediaKindsDeps = Pick<
  WindowKindsDeps,
  'tabOpeners' | 'windowTabs' | 'runs' | 'plays' | 'vaults' | 'window' | 'getTarget' | 'runCommand'
>

export function createMediaKinds(_deps: MediaKindsDeps) {
  return {}
}
