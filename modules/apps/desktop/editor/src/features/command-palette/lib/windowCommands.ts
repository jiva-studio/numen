/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { keysOf } from './chords'
import { isAlwaysOffered } from './offered'
import type { Command } from '../types'
import type { Words } from '../words'

export const windowCommandsOf = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'note',
    text: words.newNote,
    ...keysOf('note', agent),
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'deck',
    text: words.newDeck,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'importUrl',
    text: words.importUrl,
    group: 'window',
    needs: 'address',
    isOffered: (at) => at.isReady,
  },
  {
    id: 'plex',
    text: words.newPlex,
    ...keysOf('plex', agent),
    group: 'window',
    isOffered: isAlwaysOffered,
  },
  { id: 'files', text: words.files, group: 'window', isOffered: isAlwaysOffered },
  {
    id: 'agent',
    text: words.newAgent,
    ...keysOf('agent', agent),
    group: 'window',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'close',
    text: words.close,
    ...keysOf('close', agent),
    group: 'window',
    isOffered: (at) => at.tab !== '',
  },
  {
    id: 'find',
    text: words.find,
    keys: words.findKeys,
    group: 'window',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'appearance',
    text: words.appearance,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  { id: 'mode', text: words.mode, group: 'window', needs: 'choosing', isOffered: isAlwaysOffered },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'textScale',
    text: words.textScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'syncing',
    text: words.syncing,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'nesting',
    text: words.nesting,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'parts',
    text: words.parts,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'performanceProfile',
    text: words.performanceProfile,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  {
    id: 'indexingModel',
    text: words.indexingModel,
    group: 'window',
    needs: 'choosing',
    isOffered: isAlwaysOffered,
  },
  { id: 'settings', text: words.settings, group: 'window', isOffered: isAlwaysOffered },
]
