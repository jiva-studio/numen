/**
 * What the mocked application answers, and what a test sets before the window
 * draws.
 *
 * the vaults this installation holds, and `folders` what each folder holds.
 */
import type { Entry } from '@/entities/file'
import type { Tab } from '@/entities/tab'
import type { PluginSettingInfo } from '@/entities/settings'
import type { JsonObject, JsonValue } from '@/shared/api/json'

interface MockVaultState {
  isReady: boolean
  error: string
  detachError: string
  opening: string | null
  names: ReturnType<typeof nameAnswer>[]
  searchResults: (ReturnType<typeof searchResultAnswer> | ReturnType<typeof sourceAnswer>)[]
  savedTabs: { readonly tabs: readonly Tab[]; readonly activeId: string }
  detachedTab: Tab | null
  types: Record<string, string>
  tabKind: string
  listable: boolean
  isOpenRefused: boolean
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
  searchResults: [],
  /** The open tabs the core kept from before the window started. */
  savedTabs: { tabs: [], activeId: '' },
  /** The tab that readDetachedTab returns for its id. */
  detachedTab: null,
  /**
   * The note type at each path, as the vault reports it. A path without an
   * entry is an ordinary note.
   */
  types: {},
  /** The kind of tab a plugin registered for the notes, or none. */
  tabKind: '',
  /** Whether the list of vaults answers at all. */
  listable: true,
  isOpenRefused: false,
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
  return {
    isNote: true,
    type: mockVaultState.types[path] ?? ('note' as const),
    tabKind: mockVaultState.tabKind,
  }
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

/** The changes to the files of the vault that the core reports, and the stream the window receives them on. */
export const changed = (() => {
  interface Change {
    paths: string[]
    renamed: { from: string; to: string }[]
  }
  const queue: Change[] = []
  let wake: (() => void) | null = null
  async function* stream() {
    for (;;) {
      while (queue.length > 0) yield { reload: false, versionPaths: [], ...queue.shift()! }
      await new Promise<void>((woken) => {
        wake = woken
      })
    }
  }
  return {
    report: (change: Change) => {
      queue.push(change)
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
const maker = (() => {
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

/** One name the vault answers a search with, of a note of some kind, as the core sends it. */
export const nameAnswer = (path: string, title: string, type: string = 'note') => ({
  note: { path, title },
  ranges: [] as [],
  type,
  ...(type === 'note' ? {} : { tabKind: type }),
})

/** One passage the search answers with, read out of a note of some kind, as the core sends it. */
export const searchResultAnswer = (path: string, title: string, type: string = 'note') => ({
  path,
  note: { path, title },
  text: 'what it says',
  byteRange: { start: 0, end: 4 },
  matchLine: 3,
  matchRanges: [] as [],
  type,
  ...(type === 'note' ? {} : { tabKind: type }),
  selector: '',
})

/** One passage read out of a file that is no note, at a place its selector names. */
export const sourceAnswer = (path: string, selector: JsonObject) => ({
  path,
  text: 'what it says',
  byteRange: { start: 0, end: 4 },
  matchLine: 3,
  matchRanges: [] as [],
  type: 'note',
  selector: JSON.stringify(selector),
})

/** Everything answered back where it opens, which is where every test begins. */
export const forgetAnswers = () => {
  mockVaultState.isReady = true
  mockVaultState.opening = 'Root.md'
  mockVaultState.names = []
  mockVaultState.searchResults = []
  mockVaultState.savedTabs = { tabs: [], activeId: '' }
  mockVaultState.detachedTab = null
  mockVaultState.types = {}
  mockVaultState.tabKind = ''
  mockVaultState.listable = true
  mockVaultState.isOpenRefused = false
  mockVaultState.writeError = ''
  mockVaultState.app = []
  mockVaultState.detachError = ''
  mockVaultState.applied = 'preset:numen'
  mockVaultState.mode = 'system'
  mockVaultState.sizes = { interfaceScale: 1, textScale: 1 }
  maker.forget()
  outside.forget()
  changed.forget()
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
  const values: Record<string, JsonValue> = {
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
export const keepSetting = (path: string, value: JsonValue): void => {
  if (path === 'app.appearance.theme') mockVaultState.applied = String(value)
  if (path === 'app.appearance.mode') mockVaultState.mode = value as typeof mockVaultState.mode
  if (path === 'app.appearance.interfaceScale') {
    mockVaultState.sizes = { ...mockVaultState.sizes, interfaceScale: Number(value) }
  }
  if (path === 'app.appearance.textScale') {
    mockVaultState.sizes = { ...mockVaultState.sizes, textScale: Number(value) }
  }
}
