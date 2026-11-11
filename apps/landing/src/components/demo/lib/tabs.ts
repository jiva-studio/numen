import { branch, pane, type Tab, type Workspace } from '@numen/workbench'
import { BOLTZMANN_PATH } from './documents'

export type TabKind = 'files' | 'plex' | 'agent' | 'note' | 'pdf' | 'recording' | 'link' | 'deck'

export const FILES_TAB = 'files'
export const PLEX_TAB = 'plex'
export const AGENT_TAB = 'agent'
export const NOTE_TAB = 'note'
export const PDF_TAB = 'pdf'
export const RECORDING_TAB = 'recording'
export const LINK_TAB = 'link'
export const DECK_TAB = 'deck'

/** The tabs the window opens with, and the kind each one draws. */
export const DEMO_TABS: readonly (Tab & { readonly kind: TabKind })[] = [
  { id: FILES_TAB, kind: 'files', title: 'Files' },
  { id: PLEX_TAB, kind: 'plex', title: 'Entropy' },
  { id: AGENT_TAB, kind: 'agent', title: 'Agent' },
  { id: NOTE_TAB, kind: 'note', title: 'Entropy' },
  { id: PDF_TAB, kind: 'pdf', title: 'Boltzmann 1877' },
  { id: RECORDING_TAB, kind: 'recording', title: 'Lecture 4' },
  { id: LINK_TAB, kind: 'link', title: 'Feynman 1964' },
  { id: DECK_TAB, kind: 'deck', title: 'Thermodynamics' },
]

/** The paper the pdf tab shows. */
export const PDF_TAB_PATH = BOLTZMANN_PATH

/** The panes the window opens with: the files, the plex and the note tabs, and the agent. */
export function createDemoWorkspace(): Workspace {
  return {
    root: branch(
      'root',
      [
        pane('main', [FILES_TAB]),
        branch(
          'middle',
          [
            pane('map', [PLEX_TAB]),
            pane('note', [NOTE_TAB, PDF_TAB, RECORDING_TAB, LINK_TAB, DECK_TAB], NOTE_TAB),
          ],
          [0.52, 0.48],
        ),
        pane('aside', [AGENT_TAB]),
      ],
      [0.22, 0.5, 0.28],
    ),
    axis: 'horizontal',
    focus: 'map',
  }
}
