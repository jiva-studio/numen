/**
 * What the window offers while it holds no tab.
 *
 * The screen renders these rows and decides nothing, so a test can query what a
 * window on a vault, or on none, presents to the user.
 */
import type { Keystroke } from '@numen/ui'
import type { VaultRow, WelcomeAction } from '@numen/workbench'
import type { VaultList } from '@/entities/vault'
import { createRowKeys, COMMAND, COMMANDS_ROW } from '@/shared/model/command'

/** What the welcome screen is drawn over. */
export interface CurrentVault {
  readonly vault: string
  readonly isReady: boolean
}

/** Every label the welcome screen shows. */
export interface Words {
  readonly find: string
  readonly findKeys: Keystroke
  readonly commands: string
  readonly commandsKeys: Keystroke
  readonly newNote: string
  readonly settings: string
  readonly vaults: string
  readonly current: string
  readonly gone: string
  readonly newVault: string
  readonly newVaultDetail: string
}

/** The row that puts the commands up, which is no command of its own. */
export const COMMANDS = COMMANDS_ROW

/**
 * The entry points, in render order. The settings come last and are always offered.
 * The rest depend on a vault: a window showing none offers none of them.
 * A note is offered once the vault has been read, followed by the commands the plugins declare.
 * Every keystroke shown is the one the table binds, so a displayed key is a key that works.
 */
export const createWelcomeActions = (
  currentVault: CurrentVault,
  words: Words,
  agent: string,
  contributed: readonly WelcomeAction[] = [],
): readonly WelcomeAction[] => {
  const settings: WelcomeAction = {
    id: COMMAND.openSettings,
    text: words.settings,
    ...createRowKeys(COMMAND.openSettings, agent),
  }
  if (currentVault.vault === '') return [settings]
  const note: readonly WelcomeAction[] = currentVault.isReady
    ? [{ id: COMMAND.createNote, text: words.newNote, ...createRowKeys(COMMAND.createNote, agent) }]
    : []
  return [
    { id: COMMAND.searchVault, text: words.find, keys: words.findKeys },
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
