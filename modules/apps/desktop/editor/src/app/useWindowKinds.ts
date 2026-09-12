/**
 * Tab kinds and openers declared for the desktop window.
 */
import { computed, shallowRef, watch } from 'vue'
import { useConversation } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import { cards } from '@/entities/deck/cards'
import { presets } from '@/entities/deck/presets'
import { documents } from '@/widgets/document-viewer/wire'
import { books } from '@/widgets/book-reader/wire'
import { recordings } from '@/entities/media/wire'
import { running } from '@/shared/artifacts'
import { usePlexView } from '@/widgets/plex-graph/composables/usePlexView'
import { useDocumentReader } from '@/widgets/document-viewer/composables/useDocumentReader'
import { useBookReader } from '@/widgets/book-reader/composables/useBookReader'
import { WORDS as bookWords } from '@/widgets/book-reader/words'
import { CREATABLE } from '@/widgets/note-editor/maker'
import type { CommandTarget } from '@/features/command-palette/target'
import { invocationOf } from '@/features/command-palette/target'
import { does, type CommandDeps } from '@/features/command-palette/handlers'
import { lands, type DestinationDeps } from '@/features/command-palette/destination'
import type { FileOpeners } from '@/entities/tab/openers'
import { createFileCreators } from '@/entities/tab/makers'
import type { useWindowTabs } from '@/entities/tab/windowTabs'
import type { MessageLog } from '@/shared/notices/messages'
import { agentKind, useAgentConversation } from '@/widgets/agent-chat/composables/useAgentConversation'
import { documentKind, useDocumentTab } from '@/widgets/document-viewer/composables/useDocumentTab'
import { bookKind, useBookTab } from '@/widgets/book-reader/composables/useBookTab'
import { recordingKind, type MediaTabDeps } from '@/entities/media/kind'
import { RECORDINGS } from '@/widgets/media-recording/kind'
import { URLS } from '@/widgets/media-url/kind'
import { useTranscript } from '@/entities/media/transcript'
import type { createMediaTypeProbe } from '@/entities/media/player'
import { filesKind } from '@/widgets/file-manager/composables/useFilesTab'
import { useFileTree } from '@/widgets/file-manager/composables/useFileTree'
import { plexKind } from '@/widgets/plex-graph/composables/usePlexTab'
import { core as agent } from '@/widgets/agent-chat/core'
import { WORDS as talk } from '@/widgets/agent-chat/words'
import { WORDS as cardWords } from '@/entities/deck/words'
import { WORDS as words } from '@/shared/words'
import { CONVERSATION, minted } from '@/entities/tab/workspace'
import type { Source, Core } from '@/shared/core'
import type { runSupport } from '@/features/command-palette/runs'
import type { useNoteEditors } from './useNoteEditors'
import type { useSettings } from './useSettings'
import type { useVaults } from './useVaults'
import type { useWindowDisplay } from './useWindowDisplay'

export interface WindowKindsDeps {
  core: Core
  log: MessageLog
  puts: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  runs: ReturnType<typeof runSupport>
  plays: ReturnType<typeof createMediaTypeProbe>
  editing: ReturnType<typeof useNoteEditors>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
  window: ReturnType<typeof useWindowDisplay>
  where: () => CommandTarget
  carries: (id: string, target: CommandTarget) => void
  doing: () => CommandDeps
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
      await running.fetchArtifact(path)
    } catch (error) {
      told(formatErrorMessage(error), 'error')
      return
    }
    editing.notes.changed([path])
  }

  const made = createFileCreators(
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
    openDestination: (landing) => void lands(landing, places),
    runCommand: (id, paths, name, source) => {
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
    movePath: (from, to) => does(invocationOf('move', { ...where(), path: from }, to), doing(), words),
    setDraggedPaths: (paths) => {
      dragged.value = paths
    },
    createFolder: (path) => does(invocationOf('makeFolder', where(), path), doing(), words),
    createNote: async (folder) => (await editing.making.createUntitled(folder, []))?.path ?? '',
    createDeck: (folder, name) => made.makes('deck', folder, name),
    createStencil: (folder, name) => made.makes('stencil', folder, name, [cardWords.newField]),
    createPreset: (folder, name) => made.makes('preset', folder, name),
    importAddress: (folder, address) => made.imports(folder, address),
    showError: (text) => told(text, 'error'),
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
