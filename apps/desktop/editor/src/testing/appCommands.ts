/**
 * The commands of the editor as a window collects them from its features, for a test that reads
 * the rows. Nothing runs them, so the window they use does nothing.
 */
import { collectFeatureCommands } from '@/app/window/createFeatureCommands'
import type { Command, Notes } from '@/entities/command'
import { createNoteWriter } from '@/entities/note'
import { WORDS } from '@/shared/ui/words'

const noop = (): void => undefined

const NO_NOTES: Notes = {
  getTabAt: () => null,
  getPath: (id) => id,
  asking: () => false,
  flushPending: () => Promise.resolve(),
  close: noop,
  openFile: noop,
}

/** A window whose tabs hold nothing and whose core is never asked. */
export const createEmptyWindow = (): Parameters<typeof collectFeatureCommands>[0] => ({
  notes: {
    making: createNoteWriter(noop),
    notes: NO_NOTES,
    navigator: {
      travel: () => Promise.resolve(),
      leave: () => Promise.resolve(),
      getOpeningNote: () => '',
      showTab: noop,
    },
    writeMessage: noop,
  },
  files: { notes: NO_NOTES, writeMessage: noop },
  vaults: { setVaultName: noop, reload: noop, writeMessage: noop },
  tabs: {
    tabLinks: {
      tabs: [],
      windows: [],
      linkTabs: noop,
      unlinkTab: noop,
      getGroupOf: () => null,
    },
    getTabs: () => [],
    getKind: () => undefined,
    isLinkable: () => false,
    isAsking: () => false,
    detach: noop,
    move: noop,
  },
  versions: { listItems: () => Promise.resolve([]), writeMessage: noop },
})

/** Every command of the editor in the order the palette draws them. */
export const createAppCommands = (agent?: string): readonly Command[] =>
  collectFeatureCommands(createEmptyWindow(), WORDS, agent).commands
