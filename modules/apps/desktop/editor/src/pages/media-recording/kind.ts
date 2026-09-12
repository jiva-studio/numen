/** What the window keeps a recording tab by, and what draws it. */
import type { Medium } from '@/entities/media'
import { RECORDING } from '@/entities/tab'
import RecordingTab from './ui/RecordingTab.vue'

/** The recordings of the vault, played. */
export const RECORDINGS: Medium<typeof RECORDING> = {
  tab: RECORDING,
  source: 'recording',
  draws: RecordingTab,
  hands: (puts, opens) => puts.registerReader({ kind: RECORDINGS.source }, opens),
}
