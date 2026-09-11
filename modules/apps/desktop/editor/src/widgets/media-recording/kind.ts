/** What the window keeps a recording tab by, and what draws it. */
import type { Medium } from '../../entities/media/kind'
import { RECORDING } from '../../entities/tab/workspace'
import RecordingTab from './RecordingTab.vue'

/** The recordings of the vault, played. */
export const RECORDINGS: Medium<typeof RECORDING> = {
  tab: RECORDING,
  source: 'recording',
  draws: RecordingTab,
  hands: (puts, opens) => puts.reads({ kind: RECORDINGS.source }, opens),
}
