/**
 * What a window with no content offers, resolved without a screen.
 *
 * The negatives are the ones worth having: a window over no vault offers no
 * way into one, a vault that never opened offers no new note, and a key is
 * drawn on a row only where the table binds that letter to that command.
 */
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { getKeyChord } from '@numen/ui'
import type { VaultList, Vault } from '@/entities/vault'
import { ReportedError } from '@numen/result'
import {
  COMMAND,
  createRowKeys,
  findIcon,
  type CommandDeps,
  type CommandTarget,
} from '@/shared/model/command'
import { WORDS as uiWords } from '@/shared/ui/words'
import { useWelcomeScreen, type WelcomeScreenProps } from '../model/useWelcomeScreen'
import { WELCOME_WORDS } from '../words'
import {
  COMMANDS,
  createVaultRows,
  createWelcomeActions,
  type CurrentVault,
  type Words,
} from './screen'

const SETTINGS = COMMAND.openSettings
const APPLE = 'MacIntel'
const LINUX = 'Linux x86_64'

/** The screen's strings. Each word is its own value. */
const words: Words = {
  find: 'find',
  findKeys: getKeyChord('k', APPLE),
  commands: 'commands',
  commandsKeys: getKeyChord('p', APPLE),
  newNote: 'new note',
  vaults: 'vaults',
  current: 'current',
  gone: 'gone',
  readOnly: 'read only',
  settings: 'settings',
  newVault: 'new vault',
  newVaultDetail: 'choose a folder',
}

const createCurrentVault = (overrides: Partial<CurrentVault> = {}): CurrentVault => ({
  vault: 'physics',
  isReady: true,
  ...overrides,
})

const vault = (id: string, name: string, isMissing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: isMissing,
  isReadOnly: false,
})

const createVaultList = (vaults: readonly Vault[], current: string): VaultList => ({
  vaults,
  showing: current,
})

/** The commands the plugins declare over the window, as the screen is handed them. */
const CONTRIBUTED = [
  { id: 'org.example.sample:graph.run', text: 'new graph', keys: getKeyChord('x', APPLE, true) },
  { id: 'org.example.sample:chat.run', text: 'new chat' },
]

const two = createVaultList([vault('a', 'Physics'), vault('b', 'Heat')], 'a')

describe('the ways into the vault', () => {
  it('shows only settings when no vault is active', () => {
    expect(
      createWelcomeActions(createCurrentVault({ vault: '', isReady: false }), words, APPLE).map(
        (one) => one.id,
      ),
    ).toStrictEqual([SETTINGS])
  })

  it('are the settings alone where a window showing no vault says it is ready', () => {
    expect(
      createWelcomeActions(createCurrentVault({ vault: '' }), words, APPLE).map((one) => one.id),
    ).toStrictEqual([SETTINGS])
  })

  it('are those of a vault that has been read, the commands of the plugins after the note', () => {
    const ways = createWelcomeActions(createCurrentVault(), words, APPLE, CONTRIBUTED)
    expect(ways.map((one) => one.id)).toStrictEqual([
      COMMAND.searchVault,
      COMMANDS,
      COMMAND.createNote,
      'org.example.sample:graph.run',
      'org.example.sample:chat.run',
      SETTINGS,
    ])
  })

  it('are each called what the window calls them, and a plugin command what it is declared as', () => {
    const ways = createWelcomeActions(createCurrentVault(), words, APPLE, CONTRIBUTED)
    expect(ways.map((one) => one.text)).toStrictEqual([
      words.find,
      words.commands,
      words.newNote,
      'new graph',
      'new chat',
      words.settings,
    ])
  })

  it('are those of the window alone where no plugin declares a command over the window', () => {
    const ways = createWelcomeActions(createCurrentVault(), words, APPLE)
    expect(ways.map((one) => one.id)).toStrictEqual([
      COMMAND.searchVault,
      COMMANDS,
      COMMAND.createNote,
      SETTINGS,
    ])
  })

  /** A note cannot be made in a vault that never opened. The rest stand. */
  it('leave out the new note where the vault could not be opened', () => {
    const ways = createWelcomeActions(
      createCurrentVault({ isReady: false }),
      words,
      APPLE,
      CONTRIBUTED,
    )
    expect(ways.map((one) => one.id)).toStrictEqual([
      COMMAND.searchVault,
      COMMANDS,
      'org.example.sample:graph.run',
      'org.example.sample:chat.run',
      SETTINGS,
    ])
  })

  /** The settings are the installation's, so they are offered last of all. */
  it('offer the settings last, on a vault and off one', () => {
    expect(createWelcomeActions(createCurrentVault(), words, APPLE).at(-1)?.id).toBe(SETTINGS)
    expect(createWelcomeActions(createCurrentVault({ vault: '' }), words, APPLE).at(-1)?.id).toBe(
      SETTINGS,
    )
  })
})

describe('the keystroke drawn on a way', () => {
  const getKeys = (id: string, agent = APPLE) =>
    createWelcomeActions(createCurrentVault(), words, agent, CONTRIBUTED).find(
      (one) => one.id === id,
    )?.keys

  it('is the one the table binds to that command', () => {
    expect(getKeys(COMMAND.createNote)).toStrictEqual(createRowKeys(COMMAND.createNote, APPLE).keys)
    expect(getKeys('org.example.sample:graph.run')).toStrictEqual(CONTRIBUTED[0]?.keys)
  })

  it('is the key the keyboard in hand holds', () => {
    expect(getKeys(COMMAND.createNote, LINUX)).toStrictEqual({ icons: ['control'], letter: 'N' })
  })

  it('is the two the window keeps for itself, as the window passes them over', () => {
    expect(getKeys(COMMAND.searchVault)).toBe(words.findKeys)
    expect(getKeys(COMMANDS)).toBe(words.commandsKeys)
  })
})

describe('the vaults the screen lists', () => {
  it('holds every vault on the list, under its name', () => {
    expect(createVaultRows(two, words).map((one) => one.name)).toStrictEqual(['Physics', 'Heat'])
  })

  it('marks active item and renders normally', () => {
    expect(createVaultRows(two, words)[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.current,
    })
  })

  it('marks one whose folder is not there', () => {
    const target = createVaultRows(createVaultList([vault('b', 'Heat', true)], 'a'), words)
    expect(target[0]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
      detail: words.gone,
    })
  })

  it('says the folder is gone of the vault it is showing, which is the worse news', () => {
    const target = createVaultRows(createVaultList([vault('a', 'Physics', true)], 'a'), words)
    expect(target[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.gone,
    })
  })

  it('marks a vault another plugin manages as read only, besides the mark of the vault it is showing', () => {
    const managed = { ...vault('a', 'Food'), isReadOnly: true }
    const target = createVaultRows(createVaultList([managed, { ...managed, id: 'b' }], 'a'), words)
    expect(target.map((one) => one.detail)).toStrictEqual([
      `${words.current}, ${words.readOnly}`,
      words.readOnly,
    ])
  })

  it('says nothing alongside a vault that is neither', () => {
    expect(createVaultRows(two, words)[1]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
    })
  })

  it('is empty where the installation holds no vault', () => {
    expect(createVaultRows(createVaultList([], ''), words)).toStrictEqual([])
  })
})

describe('useWelcomeScreen', () => {
  const CustomIcon = defineComponent({ template: '<svg />' })

  const createTarget = (vaultId = 'a', isReady = true): CommandTarget => ({
    tab: '',
    kind: null,
    path: '',
    title: '',
    file: '',
    vault: { id: vaultId, name: 'Physics' },
    isReady,
  })

  const createCommandDeps = (overrides: Partial<CommandDeps> = {}): CommandDeps => ({
    notes: {
      getTabAt: () => null,
      getPath: () => '',
      isAsking: () => false,
      flushPending: vi.fn(() => Promise.resolve()),
      close: vi.fn(),
      openFile: vi.fn(),
    },
    navigator: {
      openTab: vi.fn(),
      openSettings: vi.fn(),
      closeTab: vi.fn(),
      search: vi.fn(),
    },
    runSupport: {
      canRun: () => true,
      reportCannotRun: vi.fn(),
    },
    registry: {
      get: () => undefined,
    },
    settings: {
      chooseSetting: vi.fn(() => Promise.resolve()),
    },
    writeMessage: vi.fn(),
    ...overrides,
  })

  const mountWelcome = (overrides: Partial<WelcomeScreenProps> = {}) => {
    const searchOpen = ref(false)
    const paletteOpen = ref(false)
    const props: WelcomeScreenProps = {
      vaultList: two,
      tabs: [],
      commands: {
        palette: { open: paletteOpen, setOpen: vi.fn() },
      } as unknown as WelcomeScreenProps['commands'],
      search: { open: searchOpen, setOpen: vi.fn() } as unknown as WelcomeScreenProps['search'],
      doing: createCommandDeps(),
      getTarget: () => createTarget(),
      pluginWays: [
        { id: 'plugin.custom', text: 'Custom Way', icon: CustomIcon },
        { id: 'plugin.plain', text: 'Plain Way' },
      ],
      runCommand: vi.fn(),
      agent: LINUX,
      ...overrides,
    }
    let state: ReturnType<typeof useWelcomeScreen> | undefined
    const Host = defineComponent({
      setup() {
        state = useWelcomeScreen(props)
        return () => null
      },
    })
    const wrapper = mount(Host)
    if (!state) throw new Error('useWelcomeScreen did not initialize')
    return { screen: state, props, searchOpen, paletteOpen, wrapper }
  }

  it('builds adding footer and ways with icons and words', () => {
    const { screen, wrapper } = mountWelcome()

    expect(screen.adding.value.text).toBe(uiWords.newVault)
    expect(screen.adding.value.detail).toBe(WELCOME_WORDS.newVaultDetail)
    expect(screen.adding.value.icon).toBe(findIcon(COMMAND.createVault))
    expect(screen.adding.value.icon).toBeDefined()

    const ways = screen.ways.value
    expect(ways.find((w) => w.id === COMMAND.searchVault)?.text).toBe(uiWords.find)
    expect(ways.find((w) => w.id === COMMAND.searchVault)?.icon).toBe(findIcon(COMMAND.searchVault))
    expect(ways.find((w) => w.id === COMMAND.createNote)?.text).toBe(uiWords.newNote)
    expect(ways.find((w) => w.id === 'plugin.custom')?.icon).toBe(CustomIcon)
    const plain = ways.find((w) => w.id === 'plugin.plain')
    expect(plain).toBeDefined()
    expect('icon' in (plain ?? {})).toBe(false)

    wrapper.unmount()
  })

  it('handles onOpen for unknown, current, and other vaults and reports invocation errors', async () => {
    const openCurrentVault = vi.fn()
    const openHandler = vi.fn()
    const doing = createCommandDeps({
      registry: {
        get: (id) => (id === COMMAND.openVault ? openHandler : undefined),
      },
    })
    const { screen, wrapper } = mountWelcome({ doing, openCurrentVault })

    // Unknown vault id does nothing
    screen.onOpen('unknown')
    expect(openCurrentVault).not.toHaveBeenCalled()
    expect(openHandler).not.toHaveBeenCalled()

    // Current vault id calls openCurrentVault and not openVault command
    screen.onOpen('a')
    expect(openCurrentVault).toHaveBeenCalledOnce()
    expect(openHandler).not.toHaveBeenCalled()

    // Other vault id invokes COMMAND.openVault
    screen.onOpen('b')
    await flushPromises()
    expect(openHandler).toHaveBeenCalledOnce()

    // Registry throwing reports error to doing.writeMessage
    const failingDoing = createCommandDeps({
      registry: {
        get: () => {
          throw new ReportedError('registry failure')
        },
      },
    })
    const failing = mountWelcome({ doing: failingDoing })
    failing.screen.onOpen('b')
    await flushPromises()
    expect(failingDoing.writeMessage).toHaveBeenCalledWith('registry failure', 'error')

    failing.wrapper.unmount()
    wrapper.unmount()
  })

  it('handles window keydown shortcuts only while welcoming', async () => {
    const openCurrentVault = vi.fn()
    const { paletteOpen, wrapper } = mountWelcome({ openCurrentVault })

    // Non-shortcut key does not preventDefault or open
    const plainKey = new KeyboardEvent('keydown', { key: 'x', cancelable: true })
    window.dispatchEvent(plainKey)
    expect(plainKey.defaultPrevented).toBe(false)
    expect(openCurrentVault).not.toHaveBeenCalled()

    // When command palette is open, shortcut is ignored
    paletteOpen.value = true
    const blockedShortcut = new KeyboardEvent('keydown', {
      key: 'a',
      cancelable: true,
    })
    window.dispatchEvent(blockedShortcut)
    expect(blockedShortcut.defaultPrevented).toBe(false)
    expect(openCurrentVault).not.toHaveBeenCalled()

    // When welcoming, A prevents default and opens vault 0 ('a')
    paletteOpen.value = false
    const activeShortcut = new KeyboardEvent('keydown', {
      key: 'a',
      cancelable: true,
    })
    window.dispatchEvent(activeShortcut)
    expect(activeShortcut.defaultPrevented).toBe(true)
    expect(openCurrentVault).toHaveBeenCalledOnce()

    wrapper.unmount()
  })
})
