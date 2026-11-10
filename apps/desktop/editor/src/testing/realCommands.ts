/**
 * A window of commands built from the real pieces: the plugin host with the slots and keybindings
 * the manifests of the plugins declare, the palette model, the runner and the handler table. Only
 * the edge is a double: the navigation of a tab and the events the core is notified of.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import {
  createPluginHost,
  createWhenReader,
  getWhenValues,
  type DeclaredSlotEntry,
  type PluginHost,
} from '@numen/plugin-loader'
import { runSupport } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import { createSlotCommandFinder } from '@/app/commands/pluginSlots'
import { useCommands, type CommandsDepsOptions } from '@/app/commands/useCommands'
import { WORDS } from '@/shared/words'
import { createAppCommands } from './appCommands'

export interface ManifestKeybinding {
  readonly key: string
  readonly command: string
  readonly values?: Readonly<Record<string, string>>
  readonly when?: string
}

export interface ManifestSlot {
  readonly slot: string
  readonly command?: string
  readonly values?: Readonly<Record<string, string>>
}

export interface PluginManifest {
  readonly id: string
  readonly contributes: {
    readonly slots?: readonly (ManifestSlot & Record<string, unknown>)[]
    readonly keybindings?: readonly ManifestKeybinding[] | undefined
  }
}

const PLUGINS_DIRECTORY = resolve(import.meta.dirname, '../../../../../plugins')

/** The manifest of the plugin in the folder, read from its JSON file. */
export const readManifest = (folder: string): PluginManifest =>
  JSON.parse(readFileSync(resolve(PLUGINS_DIRECTORY, folder, 'manifest.json'), 'utf8')) as PluginManifest

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

const createHost = (parts: RealWindowParts): PluginHost => {
  const whenReader = createWhenReader()
  const getValues = (target: Parameters<typeof getWhenValues>[0]['target']) =>
    getWhenValues({ window: windowValues, target, contentTypes: {} })
  const entries = toSlotEntries(parts.manifests)
  const host = createPluginHost({
    registerKind: () => () => undefined,
    registerExtensionReader: () => () => undefined,
    openTab: () => Promise.resolve('tab-1'),
    getSlotWhenValues: getValues,
    getSlotEntries: () => entries,
    findSlotCommand: createSlotCommandFinder({
      getPluginCommands: () => [],
      getDeclaredCommands: () => [],
      getEditorCommands: () => createAppCommands(),
      getWhenValues: getValues,
      whenReader,
    }),
  })
  for (const manifest of parts.manifests) {
    host.registerKeybindings(manifest.id, manifest.contributes.keybindings ?? [])
  }
  return host
}

const createCore = (emitEvent: RealWindowParts['emitEvent']) => {
  const base = {
    fileKinds: (paths: readonly string[]) =>
      Promise.resolve(new Map(paths.map((one) => [one, { type: 'note' }]))),
    ...(emitEvent ? { emitEvent } : {}),
  }
  return new Proxy<Record<string, unknown>>(base, {
    get: (target, name: string) => target[name] ?? (() => Promise.resolve([])),
  })
}

const toDeps = (options: object): CommandsDepsOptions => options as CommandsDepsOptions

/** The commands of a window over the note, whose tab navigation is `parts.navigateTab`. */
export function createRealWindow(parts: RealWindowParts) {
  const messages: string[] = []
  const pluginHost = createHost(parts)
  const options = {
    core: createCore(parts.emitEvent),
    words: WORDS,
    windowTabs: {
      onKeyPress: () => false,
      navigateTab: parts.navigateTab,
      openTabOfKind: () => Promise.resolve(),
      requestClose: () => undefined,
      tabs: { value: [] },
      getTab: () => undefined,
    },
    getTarget: () => NOTE,
    knows: { getTitle: () => '', getTabAt: () => null },
    paletteLists: { getStepGroups: () => [], previewItem: () => undefined },
    runs: runSupport(),
    coverage: () => ({}),
    reached: {
      getTabAt: () => null,
      getPath: (id: string) => id,
      asking: () => false,
      flushPending: () => Promise.resolve(),
      close: () => undefined,
      openFile: () => undefined,
    },
    rest: {},
    writeMessage: (text: string) => messages.push(text),
    pluginHost,
    slots: pluginHost.slots,
  }
  return { pluginHost, messages, used: useCommands(toDeps(options)) }
}
