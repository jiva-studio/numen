/**
 * A window of commands built from the real pieces: the plugin host with the slots and keybindings
 * the manifests of the plugins declare, the palette model, the runner and the handler table. Only
 * the edge is a double: the navigation of a tab and the events the core is notified of.
 */
import { ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { createPluginHost, type DeclaredSlotEntry, type PluginHost } from '@numen/plugin-loader'
import {
  createCommandDeps,
  createSearchRequests,
  PALETTE_WORDS,
  runSupport,
  useSearch,
  type NoteLookup,
  type PaletteLists,
  type SearchDeps,
} from '@/features/command-palette'
import {
  createCommandRegistry,
  type CommandRegistry,
  type CommandTarget,
  type Notes,
  type RunSupport,
} from '@/shared/model/command'
import type { Command } from '@/entities/command'
import type { SettingsStore } from '@/entities/settings'
import type { WindowTabs } from '@/entities/tab'
import type { IndexCoverage } from '@/shared/model/notices/coverage'
import type { MessageWriter } from '@/shared/model/notices/messages'
import {
  createCommandNoteReader,
  createPluginCommandPort,
  createSearchMenu,
  createWindowSlots,
  useDeclaredCommands,
  usePluginCommands,
} from '@/features/plugin-commands'
import {
  createCommandRuns,
  createCommandSources,
  createPaletteCommands,
  createPaletteSwitching,
} from '@/app/window/createCommands'
import { createAppKeyRunner, createKeyHandler, syncKeybindings } from '@/features/hotkeys'
import { collectFeatureCommands } from '@/app/window/createFeatureCommands'
import { WORDS } from '@/shared/ui/words'
import { createNoteWriter } from '@/entities/note'
import { createAppCommands } from './appCommands'

export interface ManifestKeybinding {
  readonly key: string
  readonly command: string
  readonly values?: Readonly<Record<string, string>>
  readonly when?: string
}

interface ManifestSlot {
  readonly slot: string
  readonly command?: string
  readonly values?: Readonly<Record<string, string>>
}

export interface PluginManifest {
  readonly id: string
  readonly contributes: {
    readonly slots?: readonly (ManifestSlot & { readonly icon?: string })[]
    readonly keybindings?: readonly ManifestKeybinding[] | undefined
  }
}

const PLUGINS_DIRECTORY = resolve(import.meta.dirname, '../../../../../plugins')

/** The manifest of the plugin in the folder, read from its JSON file. */
export const readManifest = (folder: string): PluginManifest =>
  JSON.parse(
    readFileSync(resolve(PLUGINS_DIRECTORY, folder, 'manifest.json'), 'utf8'),
  ) as PluginManifest

/** The manifests of every plugin of the repository. */
export const readAllManifests = (): readonly PluginManifest[] =>
  readdirSync(PLUGINS_DIRECTORY, { withFileTypes: true })
    .filter((one) => one.isDirectory())
    .flatMap((one) => {
      try {
        return [readManifest(one.name)]
      } catch {
        return []
      }
    })

/** The note every run is over; its name is invented. */
export const NOTE: CommandTarget = {
  tab: '',
  kind: null,
  path: 'Tides.md',
  file: 'Tides.md',
  title: 'Tides',
  vault: { id: 'sea', name: 'Sea' },
  isReady: true,
}

export interface RealWindowParts {
  readonly manifests: readonly PluginManifest[]
  readonly navigateTab: (kind: string, path: string) => Promise<void>
  readonly emitEvent?: (event: string) => void
}

/** The values of the window a condition reads; a test sets them before a key. */
export const windowValues = { 'window.isFieldFocused': false, 'window.isPaletteOpen': false }

const toSlotEntries = (manifests: readonly PluginManifest[]): DeclaredSlotEntry[] =>
  manifests.flatMap((manifest) =>
    (manifest.contributes.slots ?? []).map((one) => {
      const { icon: _icon, ...entry } = one
      return { pluginId: manifest.id, ...entry }
    }),
  )

const createHost = (
  parts: RealWindowParts,
  declaredCommands: ReturnType<typeof useDeclaredCommands>,
): PluginHost => {
  const entries = toSlotEntries(parts.manifests)
  const slots = createWindowSlots({
    declaredCommands,
    facts: { getAppCommands: () => createAppCommands(), getValues: () => windowValues },
    writeIssue: () => undefined,
  })
  const host = createPluginHost({
    ...slots.hostDeps,
    registerKind: () => () => undefined,
    registerExtensionReader: () => () => undefined,
    openTab: () => Promise.resolve('tab-1'),
    getSlotEntries: () => entries,
  })
  for (const manifest of parts.manifests) {
    host.keybindings.registerKeybindings(manifest.id, manifest.contributes.keybindings ?? [])
  }
  return host
}

const createCore = () => {
  const base = {
    fileKinds: (paths: readonly string[]) =>
      Promise.resolve(new Map(paths.map((one) => [one, { type: 'note' }]))),
  }
  return new Proxy<Record<string, (...args: never[]) => Promise<object> | void>>(base, {
    get: (target, name: string) => target[name] ?? (() => Promise.resolve([])),
  })
}

const toOptions = (options: object): ComposeOptions => options as ComposeOptions

/** What the commands of the test window are made from. */
interface ComposeOptions {
  readonly core: SearchDeps
  readonly words: typeof WORDS
  readonly windowTabs: WindowTabs
  readonly getTarget: () => CommandTarget
  readonly knows: NoteLookup
  readonly paletteLists: PaletteLists
  readonly runs: RunSupport
  readonly registry: CommandRegistry
  readonly coverage: () => IndexCoverage
  readonly reached: Notes
  readonly settings: Pick<SettingsStore, 'read' | 'state'>
  readonly writeMessage: MessageWriter
  readonly pluginHost: PluginHost
  readonly settingPalette: {
    getCommands(): readonly Command[]
    choose(command: string, item: string): Promise<void>
  }
  readonly declaredCommands: ReturnType<typeof useDeclaredCommands>
  readonly readCommandNote: ReturnType<typeof createCommandNoteReader>
  readonly emitEvent?: ((event: string) => void) | undefined
}

function composeCommands(options: ComposeOptions, appCommands: readonly Command[]) {
  const { pluginHost, registry, declaredCommands, writeMessage, settings, windowTabs, getTarget } =
    options
  const plugin = usePluginCommands(registry, options.readCommandNote, pluginHost, {
    context: declaredCommands.whenContext,
    getHandlers: () => declaredCommands.contributions.handlers.value,
  })
  const searchRequests = createSearchRequests()
  const commandDeps = createCommandDeps({
    windowTabs: options.windowTabs,
    reached: options.reached,
    settingPalette: options.settingPalette,
    runs: options.runs,
    registry,
    writeMessage,
    search: searchRequests.request,
    emitEvent: options.emitEvent,
  })
  const palette = useSearch(
    options.core,
    { ...options.words, ...PALETTE_WORDS },
    { coverage: options.coverage, menu: createSearchMenu(pluginHost.slots), getTarget },
  )
  const sources = createCommandSources({
    appCommands,
    settingPalette: options.settingPalette,
    declaredCommands,
    pluginHost,
    plugin,
    getTarget,
    commandDeps,
  })
  const commands = createPaletteCommands({
    core: options.core,
    words: options.words,
    knows: options.knows,
    getTarget,
    paletteLists: options.paletteLists,
    runs: options.runs,
    sources,
  })
  const paletteSwitch = createPaletteSwitching({
    declaredCommands,
    writeMessage,
    settings,
    commands,
    search: palette,
    requests: searchRequests,
  })
  const pluginCommands = createPluginCommandPort(pluginHost, plugin.getCommandNote)
  const runs = createCommandRuns({
    commands,
    paletteSwitch,
    pluginCommands,
    commandDeps,
    writeMessage,
    getTarget,
  })
  syncKeybindings(
    pluginHost.keybindings,
    declaredCommands.contributions.keybindings,
    settings.state.plugins,
  )
  const onKeyDown = createKeyHandler({
    windowTabs,
    handleShortcut: (event, runApp) => pluginCommands.handleShortcut(event, getTarget, runApp),
    runApp: createAppKeyRunner({
      toggleSearch: paletteSwitch.toggleSearch,
      toggleCommands: paletteSwitch.toggleCommands,
      runCommand: runs.runCommand,
      runWithValues: runs.runWithValues,
      getTarget,
    }),
  })
  return {
    palette,
    commands,
    commandDeps,
    pluginRuns: { items: sources.items, runCommandToEnd: runs.runCommandToEnd },
    runChoice: runs.runChoice,
    onKeyDown,
  }
}

/** The commands of a window over the note, whose tab navigation is `parts.navigateTab`. */
export function createRealWindow(parts: RealWindowParts) {
  const messages: string[] = []
  const registry = createCommandRegistry()
  const declaredCommands = useDeclaredCommands({
    registry,
    list: () => Promise.resolve({ commands: [], slots: [] }),
  })
  const pluginHost = createHost(parts, declaredCommands)
  const options = {
    core: createCore(),
    emitEvent: parts.emitEvent,
    words: WORDS,
    windowTabs: {
      panes: { onKeyPress: () => false, requestClose: () => undefined },
      tabs: {
        navigateTab: parts.navigateTab,
        openTabOfKind: () => Promise.resolve(),
        tabs: { value: [] },
        getTab: () => undefined,
      },
    },
    getTarget: () => NOTE,
    knows: { getTitle: () => '', getTabAt: () => null },
    paletteLists: { getStepGroups: () => [], previewItem: () => undefined },
    runs: runSupport(),
    registry,
    coverage: () => ({}),
    reached: {
      getTabAt: () => null,
      getPath: (id: string) => id,
      asking: () => false,
      flushPending: () => Promise.resolve(),
      close: () => undefined,
      openFile: () => undefined,
    },
    settings: { state: { plugins: ref([]) }, read: { getCommittedSetting: () => undefined } },
    writeMessage: (text: string) => messages.push(text),
    pluginHost,
    settingPalette: { getCommands: () => [], choose: () => Promise.resolve() },
    declaredCommands,
    readCommandNote: createCommandNoteReader({ fileKinds: () => Promise.resolve(new Map()) }),
  }
  const noLinks = {
    tabs: [],
    windows: [],
    linkTabs: () => undefined,
    unlinkTab: () => undefined,
    getGroupOf: () => null,
  }
  const features = collectFeatureCommands(
    {
      notes: {
        making: createNoteWriter(() => undefined),
        notes: options.reached,
        navigator: {
          travel: () => Promise.resolve(),
          leave: () => Promise.resolve(),
          getOpeningNote: () => '',
          showTab: (kind, path) => runInBackground(parts.navigateTab(kind, path), () => undefined),
        },
        writeMessage: options.writeMessage,
      },
      files: { notes: options.reached, writeMessage: options.writeMessage },
      vaults: {
        tabsHold: { pause: () => Promise.resolve(), resume: () => undefined },
        setVaultName: () => undefined,
        reload: () => undefined,
        writeMessage: options.writeMessage,
      },
      tabs: {
        tabLinks: noLinks,
        getTabs: () => [],
        getKind: () => undefined,
        isLinkable: () => false,
        isAsking: () => false,
        detach: () => undefined,
        move: () => undefined,
      },
      versions: { listItems: () => Promise.resolve([]), writeMessage: options.writeMessage },
    },
    WORDS,
  )
  registry.registerAll(features.handlers)
  const used = composeCommands(toOptions(options), features.commands)
  return { pluginHost, messages, used }
}
