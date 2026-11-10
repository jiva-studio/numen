/**
 * The commands over the vaults an installation holds: go to a note by name, switch to another vault,
 * add, rename and remove one, and empty the trash of the vault.
 *
 * A vault is not a note. It holds the user's folder, and each of these commands moves the window to
 * a different location than it was at.
 */
import {
  COMMAND,
  ROW,
  createRowKeys,
  withWhen,
  type Command,
  type CommandInvocation,
  type FeatureCommands,
  type VaultRef,
} from '@/entities/command'
import { emptyTrash } from '@/entities/file'
import { vaults } from '@/entities/vault'
import type { MessageWriter } from '@/shared/notices/messages'
import { VAULT_WORDS } from '../words'
import { openAfterSavingTabs, type TabsHold } from './leaveVault'

/** What the commands over the vaults use of the window. */
export interface VaultCommandsDeps {
  readonly tabsHold?: TabsHold | undefined
  /** Updates the displayed name of one vault in the list. */
  readonly setVaultName: (vault: VaultRef) => void
  /**
   * The page drawn again, on the vault the window shows now. Every tab and every plex belonged to
   * the vault that has gone.
   */
  readonly reload: () => void
  readonly writeMessage: MessageWriter
}

/**
 * Another vault under this window. What the page holds belongs to the vault that has gone, so the
 * page is drawn again on the one that arrived.
 */
const showVault = async (id: string, deps: VaultCommandsDeps): Promise<void> => {
  if (!id) return
  const open = deps.tabsHold ? openAfterSavingTabs(vaults.open, deps.tabsHold) : vaults.open
  const error = await open(id)
  if (error) return deps.writeMessage(VAULT_WORDS.vaultErrors[error], 'error')
  deps.reload()
}

/**
 * Adds a folder chosen on this machine as a vault and opens it. When the user chooses no folder,
 * nothing happens.
 */
const addVault = async (deps: VaultCommandsDeps): Promise<void> => {
  const path = await vaults.choose(VAULT_WORDS.folder)
  if (!path) return
  const answer = await vaults.add(path, '')
  if (!answer.ok) return deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
  if (!answer.value) return
  await showVault(answer.value.id, deps)
}

/** A vault called something else. Its folder keeps the name it has on disk. */
const renameVault = async (
  invocation: CommandInvocation,
  deps: VaultCommandsDeps,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.vault.name) return
  const answer = await vaults.rename(invocation.vault.id, invocation.name)
  if (!answer.ok) return deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
  if (answer.value) deps.setVaultName({ id: answer.value.id, name: answer.value.name })
}

/**
 * A vault taken off the list. Erasing it puts the folder in the trash this machine keeps; removing
 * it leaves the folder where it is.
 */
const removeVault = async (
  invocation: CommandInvocation,
  deps: VaultCommandsDeps,
  shouldErase: boolean,
): Promise<void> => {
  const error = await vaults.remove(invocation.vault.id, shouldErase)
  if (error) deps.writeMessage(VAULT_WORDS.vaultErrors[error], 'error')
}

/** Deletes what the trash holds for good, and reports how many paths stayed. */
const emptyVaultTrash = async (deps: VaultCommandsDeps): Promise<void> => {
  const undeleted = await emptyTrash()
  if (undeleted > 0) deps.writeMessage(VAULT_WORDS.undeleted(undeleted), 'error')
}

const createRows = (agent: string): readonly Command[] => [
  {
    id: ROW.goToNote,
    command: COMMAND.openFile,
    text: VAULT_WORDS.goto,
    ...createRowKeys(ROW.goToNote, agent),
    group: 'vault',
    input: 'note',
    ...withWhen('window.isVaultReady'),
  },
  {
    id: COMMAND.openVault,
    text: VAULT_WORDS.openVault,
    group: 'vault',
    input: 'vault',
    ...withWhen(''),
  },
  {
    id: COMMAND.createVault,
    text: VAULT_WORDS.newVault,
    ...createRowKeys(COMMAND.createVault, agent),
    group: 'vault',
    ...withWhen(''),
  },
  {
    id: COMMAND.renameVault,
    text: VAULT_WORDS.renameVault,
    group: 'vault',
    input: 'name',
    ...withWhen('window.isVaultOpen'),
    getFieldText: (target) => target.vault.name,
  },
  {
    id: COMMAND.removeVault,
    text: VAULT_WORDS.removeVault,
    group: 'vault',
    input: 'vault',
    next: 'confirmation',
    ...withWhen(''),
    answers: {
      keeps: VAULT_WORDS.keepsVault,
      retainedNotice: VAULT_WORDS.retainedNotice,
      action: VAULT_WORDS.removeVaultAction,
      detail: VAULT_WORDS.stays,
    },
  },
  {
    id: COMMAND.eraseVault,
    text: VAULT_WORDS.eraseVault,
    group: 'vault',
    input: 'vault',
    next: 'retypedName',
    ...withWhen(''),
    warns: {
      action: VAULT_WORDS.erases,
      detail: VAULT_WORDS.binned,
      back: VAULT_WORDS.typeVaultBack,
    },
  },
  {
    id: COMMAND.emptyTrash,
    text: VAULT_WORDS.emptyTrash,
    group: 'vault',
    input: 'confirmation',
    ...withWhen('window.isVaultReady'),
    answers: {
      keeps: VAULT_WORDS.keepsTrash,
      retainedNotice: VAULT_WORDS.retainedNotice,
      action: VAULT_WORDS.emptiesTrash,
      detail: VAULT_WORDS.exactly,
    },
  },
]

/** The commands over the vaults. Going to a note runs the command that opens a file. */
export const createVaultCommands = (
  deps: VaultCommandsDeps,
  agent: string = navigator.userAgent,
): FeatureCommands => ({
  commands: createRows(agent),
  handlers: {
    [COMMAND.openVault]: (invocation) => showVault(invocation.vault.id, deps),
    [COMMAND.createVault]: () => addVault(deps),
    [COMMAND.renameVault]: (invocation) => renameVault(invocation, deps),
    [COMMAND.removeVault]: (invocation) => removeVault(invocation, deps, false),
    [COMMAND.eraseVault]: (invocation) => removeVault(invocation, deps, true),
    [COMMAND.emptyTrash]: () => emptyVaultTrash(deps),
  },
})
