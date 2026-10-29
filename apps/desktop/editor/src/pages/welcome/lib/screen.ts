/**
 * What the window offers while it holds no tab.
 *
 * The screen renders these rows and decides nothing, so a test can query what a
 * window on a vault, or on none, presents to the user.
 */
import type { PaletteKeys } from '@numen/ui'
import type { VaultRow, WelcomeAction } from '@numen/shell'
import type { VaultList } from '@/entities/vault'
import { createRowKeys } from '@/features/command-palette'

/** What the welcome screen is drawn over. */
export interface CurrentVault {
  readonly vault: string
  /** Whether that vault has been read and can serve requests. */
  readonly isReady: boolean
}

/** Every label the welcome screen shows. */
export interface Words {
  /** The palette and the commands, each with the keystroke that puts it up. */
  readonly find: string
  readonly findKeys: PaletteKeys
  readonly commands: string
  readonly commandsKeys: PaletteKeys
  /** What each of the ways into the vault the window names is called. */
  readonly newNote: string
  /** Everything this installation is configured as. */
  readonly settings: string
  /** The vaults, and what a row of that list carries beside its name. */
  readonly vaults: string
  readonly current: string
  readonly gone: string
  readonly newVault: string
  readonly newVaultDetail: string
}

/** The row that puts the commands up, which is no command of its own. */
export const COMMANDS = 'commands'

/** The row that opens the settings, which stand outside any vault. */
export const SETTINGS = 'settings'

/**
 * The entry points, in render order. The settings belong to the installation,
 * so they are always offered, last. The rest depend on a vault: a window
 * showing none offers none of them, and the vault list below is where the user
 * goes first. A note is offered once the vault has been read. The commands the
 * plugins declare over the window with a keystroke follow it.
 *
 * Every keystroke shown here is the one the table binds, so a displayed key is
 * a key that works.
 */
export const createWelcomeActions = (
  currentVault: CurrentVault,
  words: Words,
  agent: string,
  contributed: readonly WelcomeAction[] = [],
): readonly WelcomeAction[] => {
  const settings: WelcomeAction = {
    id: SETTINGS,
    text: words.settings,
    ...createRowKeys(SETTINGS, agent),
  }
  if (currentVault.vault === '') return [settings]
  const note: readonly WelcomeAction[] = currentVault.isReady
    ? [{ id: 'note', text: words.newNote, ...createRowKeys('note', agent) }]
    : []
  return [
    { id: 'find', text: words.find, keys: words.findKeys },
    { id: COMMANDS, text: words.commands, keys: words.commandsKeys },
    ...note,
    ...contributed,
    settings,
  ]
}

/**
 * The vaults the installation holds. The current vault is marked, as is one
 * whose folder is missing; the rest carry no mark.
 */
export const createVaultRows = (list: VaultList, words: Words): readonly VaultRow[] =>
  list.vaults.map((one) => {
    let aside = ''
    if (one.isMissing) {
      aside = words.gone
    } else if (one.id === list.showing) {
      aside = words.current
    }
    return {
      id: one.id,
      name: one.name,
      path: one.path,
      ...(aside ? { detail: aside } : {}),
    }
  })
