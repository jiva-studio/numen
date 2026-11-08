/**
 * What the mocked application answers, and what a test sets before the window
 * draws.
 *
 * the vaults this installation holds, and `folders` what each folder holds.
 */
import type { Entry } from '@/entities/file'
import type { OpenTabs, Tab } from '@/entities/tab'
import type { PluginSettingInfo } from '@/entities/settings'

interface MockVaultState {
  isReady: boolean
  error: string
  detachError: string
  opening: string | null
  names: {
    path: string
    title: string
    heading: string
    line: number
    at: []
    type: string
  }[]
  passages: {
    path: string
    title: string
    isNote: boolean
    text: string
    start: number
    length: number
    line: number
    at: []
    type: string
    selector: Record<string, unknown> | null
  }[]
  savedTabs: OpenTabs
  detachedTab: Tab | null
  types: Record<string, string>
  embedded: number
  listable: boolean
  writeError: string
  app: readonly PluginSettingInfo[]
  applied: string
  mode: 'system' | 'light' | 'dark'
  sizes: { interfaceScale: number; textScale: number }
}

/** What the mocked vault answers about itself, set before the window draws. */
export const mockVaultState: MockVaultState = {
  isReady: true,
  error: '',
  detachError: '',
  opening: 'Root.md',
  names: [],
  /** The passages the search answers with. */
  passages: [],
  /** The open tabs the core kept from before the window started. */
  savedTabs: { tabs: [], activeId: '' },
  /** The tab that readDetachedTab returns for its id. */
  detachedTab: null,
  /**
   * The note type at each path, as the vault reports it. A path without an
   * entry is an ordinary note.
   */
  types: {},
  /** How many of the spans the index holds carry a vector. */
  embedded: 0,
  /** Whether the list of vaults answers at all. */
  listable: true,
  /** Why the settings were not written, which is a size outside its bounds. */
  writeError: '',
  app: [],
  /** What the settings say the window is drawn as, which a test may set. */
  applied: 'preset:numen',
  mode: 'system',
  sizes: { interfaceScale: 1, textScale: 1 },
}

export const mockVaultList = {
  vaults: [{ id: 'physics', name: 'Physics', path: '/vaults/Physics', isMissing: false }],
  showing: 'physics',
}

export { mockVaultState as vaultState, mockVaultList as listed }

/** One command a plugin declares, as the list the core answers carries it. */
interface DeclaredAnswer {
  pluginId: string
  id: string
  title: string
  icon?: string
  scope?: 'note'
  shortcut?: string
  contentTypes: string[]
  extensions: string[]
  arguments: []
}

/** One entry a plugin places in a slot, as the list the core answers carries it. */
interface PlacementAnswer {
  pluginId: string
  slot: string
  command?: string
  view?: string
  group?: string
  order?: number
}

/** What the plugins declare and place. */
export const declared = {
  commands: [] as DeclaredAnswer[],
  slots: [] as PlacementAnswer[],
}

/** What each folder of the vault holds, as a listing answers it. */
export const folders: Record<string, readonly Entry[]> = {
  '': [
    { path: 'physics', name: 'physics', isFolder: true, isNote: false, type: 'note' },
    { path: 'Root.md', name: 'Root.md', isFolder: false, isNote: true, type: 'note' },
    { path: 'Cover.png', name: 'Cover.png', isFolder: false, isNote: false, type: 'note' },
  ],
  physics: [
    {
      path: 'physics/Entropy.md',
      name: 'Entropy.md',
      isFolder: false,
      isNote: true,
      type: 'note',
    },
    { path: 'physics/Kelvin.md', name: 'Kelvin.md', isFolder: false, isNote: true, type: 'note' },
  ],
}

/** Whether the file at a path is a note, and which type, as the vault answers it. */
export const getFileKind = (path: string) => {
  if (!/\.(md|note)$/u.test(path)) return { isNote: false, type: 'note' as const }
  return { isNote: true, type: mockVaultState.types[path] ?? ('note' as const) }
}

/** A stream that stays open, so nothing the window follows ever ends. */
export async function* pendingStream(): AsyncGenerator<never> {
  yield await new Promise<never>(() => undefined)
}

/**
 * Place focus requests that originate outside the window, and the stream the
 * window receives them on.
 */
export const outside = (() => {
  const queue: { path: string; ranges: { start: number; end: number }[]; tabId: string }[] = []
  let wake: (() => void) | null = null
  async function* stream() {
    for (;;) {
      while (queue.length > 0) yield queue.shift()!
      await new Promise<void>((woken) => {
        wake = woken
      })
    }
  }
  return {
    ask: (target: { path: string; start: number; length: number }) => {
      queue.push({
        path: target.path,
        ranges: [{ start: target.start, end: target.start + target.length }],
        tabId: '',
      })
      wake?.()
      wake = null
    },
    forget: () => queue.splice(0),
    stream,
  }
})()

/**
 * The vault's creation of a note of a plugin type: the file is named after the
 * title, and the vault chooses the extension, not the caller. The extension is
 * not markdown, so a window that builds the path itself does not match it.
 */
export const maker = (() => {
  const filed = new Set<string>()
  /** Whether the vault is reachable. */
  let reached = true
  return {
    forget: () => {
      filed.clear()
      reached = true
    },
    /** Makes the vault unreachable, so file creation fails. */
    fail: () => {
      reached = false
    },
    createFile: (title: string, folder: string) => {
      if (!reached) throw new Error('the vault could not be reached')
      const path = `${folder ? `${folder}/` : ''}${title}.note`
      if (filed.has(path)) return { ok: false as const, error: 'alreadyExists' as const }
      filed.add(path)
      return { ok: true as const, value: { path } }
    },
  }
})()

/** One name the vault answers a search with, of a note of some kind. */
export const nameAnswer = (path: string, title: string, type: string = 'note') => ({
  path,
  title,
  heading: '',
  line: -1,
  at: [] as [],
  type,
  ...(type === 'note' ? {} : { tabKind: type }),
})

/** One passage the search answers with, read out of a note of some kind. */
export const passageAnswer = (path: string, title: string, type: string = 'note') => ({
  path,
  title,
  isNote: true,
  text: 'what it says',
  start: 0,
  length: 4,
  line: 3,
  at: [] as [],
  type,
  ...(type === 'note' ? {} : { tabKind: type }),
  selector: null as Record<string, unknown> | null,
})

/** One passage read out of a file that is no note, at a place its selector names. */
export const sourceAnswer = (path: string, selector: Record<string, unknown>) => ({
  ...passageAnswer(path, ''),
  isNote: false,
  selector,
})

/** Everything answered back where it opens, which is where every test begins. */
export const forgetAnswers = () => {
  mockVaultState.isReady = true
  mockVaultState.opening = 'Root.md'
  mockVaultState.names = []
  mockVaultState.passages = []
  mockVaultState.savedTabs = { tabs: [], activeId: '' }
  mockVaultState.detachedTab = null
  mockVaultState.types = {}
  mockVaultState.embedded = 0
  mockVaultState.listable = true
  mockVaultState.writeError = ''
  mockVaultState.app = []
  mockVaultState.detachError = ''
  mockVaultState.applied = 'preset:numen'
  mockVaultState.mode = 'system'
  mockVaultState.sizes = { interfaceScale: 1, textScale: 1 }
  declared.commands = []
  declared.slots = []
  maker.forget()
  outside.forget()
  mockVaultList.vaults = [
    { id: 'physics', name: 'Physics', path: '/vaults/Physics', isMissing: false },
  ]
  mockVaultList.showing = 'physics'
}

/** The themes the mocked application lists, as the options of the setting that names the theme. */
const THEME_OPTIONS = [
  { value: 'preset:numen', title: 'numen', detail: '' },
  { value: 'mine:sea', title: 'sea', detail: '' },
]

/** A declared setting with the value in force and the options the mocked application resolves. */
export const getSettingInForce = (setting: PluginSettingInfo): PluginSettingInfo => {
  const values: Record<string, unknown> = {
    'appearance.theme': mockVaultState.applied,
    'appearance.mode': mockVaultState.mode,
    'appearance.interfaceScale': mockVaultState.sizes.interfaceScale,
    'appearance.textScale': mockVaultState.sizes.textScale,
  }
  return {
    ...setting,
    isShown: true,
    value: setting.key in values ? values[setting.key] : setting.default,
    ...(setting.optionsFrom === 'themes' ? { activeOptions: THEME_OPTIONS } : {}),
  }
}

/** The state of the mocked application after a setting was written. */
export const keepSetting = (path: string, value: unknown): void => {
  if (path === 'app.appearance.theme') mockVaultState.applied = String(value)
  if (path === 'app.appearance.mode') mockVaultState.mode = value as typeof mockVaultState.mode
  if (path === 'app.appearance.interfaceScale') {
    mockVaultState.sizes = { ...mockVaultState.sizes, interfaceScale: Number(value) }
  }
  if (path === 'app.appearance.textScale') {
    mockVaultState.sizes = { ...mockVaultState.sizes, textScale: Number(value) }
  }
}
