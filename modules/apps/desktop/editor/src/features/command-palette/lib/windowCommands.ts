/**
 * The commands offered over the window: what it opens, and how it is drawn.
 */
import { keysOf } from './chords'
import { isOnAnything } from './offered'
import type { Command } from '../types'
import type { Words } from '../words'

export const windowCommandsOf = (words: Words, agent: string): readonly Command[] => [
  {
    id: 'note',
    text: words.newNote,
    ...keysOf('note', agent),
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.ready,
  },
  {
    id: 'deck',
    text: words.newDeck,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.ready,
  },
  {
    id: 'stencil',
    text: words.newStencil,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.ready,
  },
  {
    id: 'newPreset',
    text: words.newPreset,
    group: 'window',
    needs: 'naming',
    isOffered: (at) => at.ready,
  },
  {
    id: 'importUrl',
    text: words.importUrl,
    group: 'window',
    needs: 'address',
    isOffered: (at) => at.ready,
  },
  {
    id: 'plex',
    text: words.newPlex,
    ...keysOf('plex', agent),
    group: 'window',
    isOffered: isOnAnything,
  },
  { id: 'files', text: words.files, group: 'window', isOffered: isOnAnything },
  {
    id: 'agent',
    text: words.newAgent,
    ...keysOf('agent', agent),
    group: 'window',
    isOffered: isOnAnything,
  },
  {
    id: 'close',
    text: words.close,
    ...keysOf('close', agent),
    group: 'window',
    isOffered: (at) => at.tab !== '',
  },
  { id: 'find', text: words.find, keys: words.findKeys, group: 'window', isOffered: isOnAnything },
  {
    id: 'appearance',
    text: words.appearance,
    group: 'window',
    needs: 'choosing',
    isOffered: isOnAnything,
  },
  { id: 'mode', text: words.mode, group: 'window', needs: 'choosing', isOffered: isOnAnything },
  {
    id: 'interfaceScale',
    text: words.interfaceScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isOnAnything,
  },
  {
    id: 'textScale',
    text: words.textScale,
    group: 'window',
    needs: 'choosing',
    isOffered: isOnAnything,
  },
  { id: 'syncing', text: words.syncing, group: 'window', needs: 'choosing', isOffered: isOnAnything },
  { id: 'hanging', text: words.hanging, group: 'window', needs: 'choosing', isOffered: isOnAnything },
  { id: 'parts', text: words.parts, group: 'window', needs: 'choosing', isOffered: isOnAnything },
  { id: 'settings', text: words.settings, group: 'window', isOffered: isOnAnything },
]
