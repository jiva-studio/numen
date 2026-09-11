/**
 * Tab kinds and openers declared for the desktop window.
 */
import { computed, shallowRef, watch } from 'vue'
import { useConversation } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import { cards } from '../shared/flashcards/cards'
import { presets } from '../widgets/preset-editor/core'
import { documents } from '../widgets/document-viewer/wire'
import { books } from '../widgets/book-reader/wire'
import { recordings } from '../shared/media/wire'
import { running } from '../shared/artifacts'
import { usePlexView } from '../widgets/plex-graph/view'
import { useDocumentReader } from '../widgets/document-viewer/open'
import { useBookReader } from '../widgets/book-reader/open'
import { WORDS as bookWords } from '../widgets/book-reader/words'
import { CREATABLE } from '../widgets/note-editor/maker'
import type { CommandTarget } from '../shared/command/target'
import { invocationOf } from '../shared/command/target'
import { does } from '../shared/command/handlers'
import { lands, type DestinationDeps } from '../shared/command/destination'
import type { FileOpeners } from '../shared/tabs/openers'
import { fileMakers } from '../shared/tabs/makers'
import type { useWindowTabs } from '../shared/tabs/windowTabs'
import type { MessageLog } from '../shared/notices/messages'
import { agentKind, useAgentConversation } from '../widgets/agent-chat/kind'
import { documentKind, useDocumentTab } from '../widgets/document-viewer/kind'
import { bookKind, useBookTab } from '../widgets/book-reader/kind'
import { recordingKind, type MediaTabDeps } from '../shared/media/kind'
import { RECORDINGS } from '../widgets/media-recording/kind'
import { URLS } from '../widgets/media-url/kind'
import { useTranscript } from '../shared/media/transcript'
import type { createMediaTypeProbe } from '../shared/media/player'
import { filesKind } from '../widgets/file-manager/kind'
import { useFileTree } from '../widgets/file-manager/listing'
import { plexKind } from '../widgets/plex-graph/kind'
import { core as agent } from '../widgets/agent-chat/core'
import { WORDS as talk } from '../widgets/agent-chat/words'
import { WORDS as cardWords } from '../shared/flashcards/words'
import { WORDS as words } from '../shared/words'
import { CONVERSATION, minted } from '../shared/tabs/workspace'
import type { Source, Core } from '../shared/core'
import type { runSupport } from '../shared/command/runs'
import type { useEditing } from './editing'
import type { useSettings } from './settings'
import type { useVaults } from './vaults'
import type { useWindowShowing } from './showing'
import type { CommandInvocation } from '../shared/command/target'

export interface WindowKindsDeps {
  core: Core
  log: MessageLog
  puts: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  runs: ReturnType<typeof runSupport>
  plays: ReturnType<typeof createMediaTypeProbe>
  editing: ReturnType<typeof useEditing>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
  window: ReturnType<typeof useWindowShowing>
  where: () => CommandTarget
  carries: (id: string, target: CommandTarget) => void
  doing: () => (action: CommandInvocation) => void
}

/**
 * Initializes and registers view kinds for the desktop window.
 */
export function useWindowKinds({
  core,
  log,
  puts,
  held,
  runs,
  plays,
  editing,
  settings,
  vaults,
  window,
  where,
  carries,
  doing,
}: WindowKindsDeps) {
  const told = log.under('command')
  const dragged = shallowRef<readonly string[]>([])

  const plexes = plexKind(held.handle, () => usePlexView(core), {
    makes: editing.making,
    ready: computed(() => !window.failure.value),
    hangs: settings.hungParts.hangs,
    parts: settings.hungParts.parts,
    opens: (path, title, showing, line) => void puts.opens(path, title, showing, line),
    inside: (paths) => core.headings(paths),
    asks: (text) => void agents.askQuestion(text),
    runs: (id, path, title) => carries(id, { ...where(), path, title }),
    opening: window.opening,
    first: () => window.first(),
    dragged,
    says: (text) => told(text, 'error'),
    writes: async () => (await editing.making.createUntitled('', []))?.path ?? '',
    creatable: CREATABLE,
  })

  const agents = agentKind(
    held.handle,
    () =>
      useAgentConversation(useConversation(agent, talk, minted(CONVERSATION)), {
        opens: (path, ...runs) => void puts.opensAt(path, runs),
        beside: (path) => void puts.opens(path, '', 'beside'),
        resolve: (written) => core.resolve('', written),
        unreachable: () => window.unreachable.value,
      }),
    () => {
      const path = plexes.looking()
      return { path, title: plexes.names(path) || path }
    },
  )

  const read = documentKind(held.handle, (path) => useDocumentTab(useDocumentReader(documents, path)), puts)

  const turned = bookKind(
    held.handle,
    (path) => useBookTab(useBookReader(books, path, bookWords, log.under('book'))),
    puts,
  )

  const over = (source: Source): MediaTabDeps => ({
    runs: (id, path, called) =>
      carries(id, {
        ...where(),
        path: '',
        title: called,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
      }),
    canRun: (run) => runs.canRun(run),
  })

  const recorded = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('recording'),
    puts,
    RECORDINGS,
  )

  const pointed = recordingKind(
    held.handle,
    (path) => useTranscript(recordings, path, { plays }),
    over('url'),
    puts,
    URLS,
  )

  watch(window.tasks, () => {
    recorded.ticked(window.tasks.value)
    pointed.ticked(window.tasks.value)
  })

  const places: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    opensAt: (path, run) => puts.opensAt(path, [run]),
    opens: (path, title, line) => void puts.opens(path, title, 'here', line),
  }

  const fetches = async (path: string): Promise<void> => {
    try {
      await running.fetches(path)
    } catch (error) {
      told(formatErrorMessage(error), 'error')
      return
    }
    editing.notes.changed([path])
  }

  const made = fileMakers(
    {
      createDeck: (title, folder) => cards.createDeck(title, folder),
      createStencil: (title, folder, fields) => cards.createStencil(title, folder, fields),
      createPreset: (title, folder) => presets.makes(title, folder),
      createUrl: async (address, folder) => {
        const made = await core.createUrl(address, folder)
        if (made.path) void fetches(made.path)
        return made
      },
    },
    puts,
    { errors: words.errors },
    told,
  )

  const files = filesKind(held.handle, () => useFileTree(core), {
    lands: (landing) => void lands(landing, places),
    runs: (id, paths, name, source) => {
      const path = paths[0] ?? ''
      carries(id, {
        ...where(),
        path,
        title: name,
        file: path,
        source,
        made: vaults.makes.value.get(path) ?? {},
        others: paths.slice(1),
      })
    },
    moves: (from, to) => does(invocationOf('move', { ...where(), path: from }, to), doing(), words),
    drags: (paths) => {
      dragged.value = paths
    },
    makes: (path) => does(invocationOf('makeFolder', where(), path), doing(), words),
    writes: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    decks: (folder, name) => made.makes('deck', folder, name),
    stencils: (folder, name) => made.makes('stencil', folder, name, [cardWords.newField]),
    presets: (folder, name) => made.makes('preset', folder, name),
    imports: (folder, address) => made.imports(folder, address),
    says: (text) => told(text, 'error'),
    canRun: (run) => runs.canRun(run),
  })

  held.declares([
    ...editing.kinds,
    ...settings.kinds,
    plexes.kind,
    agents.kind,
    read.kind,
    turned.kind,
    recorded.kind,
    pointed.kind,
    files.kind,
  ])

  return {
    plexes,
    agents,
    read,
    turned,
    recorded,
    pointed,
    files,
    made,
    places,
    dragged,
  }
}
