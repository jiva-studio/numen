/**
 * What the window opens with, and what its tabs are called.
 *
 * The workspace takes identities and gives them back; which component each one
 * stands for is settled in `App.vue`.
 */
import { branch, group, type TabLabel, type WorkspaceLayout } from '@numen/ui'

export const PLEX = 'plex'
export const AGENT = 'agent'

export const TABS: readonly TabLabel[] = [
  { id: PLEX, title: 'Plex' },
  { id: AGENT, title: 'Agent' },
]

/** The plex with the room, and the agent along the trailing edge. */
export const opening = (): WorkspaceLayout => ({
  root: branch('root', [group('main', [PLEX]), group('aside', [AGENT])], [0.72, 0.28]),
  axis: 'horizontal',
  focus: 'main',
})
