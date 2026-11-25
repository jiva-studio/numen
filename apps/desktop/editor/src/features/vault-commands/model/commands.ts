/**
 * The commands over the vaults an installation holds: go to a note by name, switch to another vault,
 * add, rename and remove one, and empty the trash of the vault.
 *
 * A vault is not a note. It holds the user's folder, and each of these commands moves the window to
 * a different location than it was at.
 */
import { type Command, type FeatureCommands } from '@/entities/command'
import {
  COMMAND,
  ROW,
  createRowKeys,
  withWhen,
  type CommandInvocation,
  type VaultRef,
} from '@/shared/model/command'
import {
  findCloudRemoteId,
  type CloudVaults,
  type RemoteVaults,
  type Vaults,
} from '@/entities/vault'
import type { PickerHost } from '@numen/plugin-loader'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { VAULT_WORDS } from '../words'
import { openCloudVault } from './openCloudVault'
import { openAfterSavingTabs, type TabsHold } from './leaveVault'

/** What getting a cloud vault uses: the vaults known, the calls that get one, and the question of where. */
export interface CloudVaultsDeps {
  readonly known: Pick<CloudVaults, 'all' | 'refresh'>
  readonly remote: Pick<RemoteVaults, 'suggestPath' | 'open'>
  readonly ask: Pick<PickerHost, 'selectOption'>
}

/**
 * What the commands over the vaults use of the window. `tabsHold` holds the writes of the open tabs
 * while the window changes vault, so a write cannot reach the vault it switches to. `reload` draws
 * the page again on the vault the window shows now, because what the page holds belongs to the
 * vault that has gone. `setVaultName` updates the displayed name of one vault in the list.
 * `cloud` is what getting a cloud vault onto this device uses. `clearTrash` returns how many files it
 * could not delete.
 */
export interface VaultCommandsDeps {
  readonly vaults: Vaults
  readonly cloud: CloudVaultsDeps
  readonly clearTrash: () => Promise<number>
  readonly tabsHold?: TabsHold | undefined
  readonly setVaultName: (vault: VaultRef) => void
  readonly reload: () => void
  readonly writeMessage: MessageWriter
}

const showVault = async (id: string, deps: VaultCommandsDeps): Promise<void> => {
  if (!id) return
  const open = deps.tabsHold
    ? openAfterSavingTabs(deps.vaults.open, deps.tabsHold)
    : deps.vaults.open
  const answer = await open(id)
  if (!answer.ok) return deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
  deps.reload()
}

/** Shows a vault of the list, or gets the cloud vault the id names onto this device. */
const openVault = async (id: string, deps: VaultCommandsDeps): Promise<void> => {
  const remoteId = findCloudRemoteId(id)
  if (!remoteId) return showVault(id, deps)
  const vault = deps.cloud.known.all.value.find((one) => one.remoteId === remoteId)
  if (!vault) return
  await openCloudVault(vault, {
    vaults: deps.vaults,
    remote: deps.cloud.remote,
    ask: deps.cloud.ask,
    writeMessage: deps.writeMessage,
    showVault: (shown) => showVault(shown, deps),
  })
  await deps.cloud.known.refresh()
}

const addVault = async (deps: VaultCommandsDeps): Promise<void> => {
  const path = await deps.vaults.choose(VAULT_WORDS.folder)
  if (!path) return
  const answer = await deps.vaults.add(path, '')
  if (!answer.ok) return deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
  if (!answer.value) return
  await showVault(answer.value.id, deps)
}

const renameVault = async (
  invocation: CommandInvocation,
  deps: VaultCommandsDeps,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.target.vault.name) return
  const answer = await deps.vaults.rename(invocation.target.vault.id, invocation.name)
  if (!answer.ok) return deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
  if (answer.value) deps.setVaultName({ id: answer.value.id, name: answer.value.name })
}

const removeVault = async (
  invocation: CommandInvocation,
  deps: VaultCommandsDeps,
  shouldErase: boolean,
): Promise<void> => {
  const answer = await deps.vaults.remove(invocation.target.vault.id, shouldErase)
  if (!answer.ok) deps.writeMessage(VAULT_WORDS.vaultErrors[answer.error], 'error')
}

const clearVaultTrash = async (deps: VaultCommandsDeps): Promise<void> => {
  const undeleted = await deps.clearTrash()
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
export const createVaultCommands = (deps: VaultCommandsDeps, agent: string): FeatureCommands => ({
  commands: createRows(agent),
  handlers: {
    [COMMAND.openVault]: (invocation) => openVault(invocation.target.vault.id, deps),
    [COMMAND.createVault]: () => addVault(deps),
    [COMMAND.renameVault]: (invocation) => renameVault(invocation, deps),
    [COMMAND.removeVault]: (invocation) => removeVault(invocation, deps, false),
    [COMMAND.eraseVault]: (invocation) => removeVault(invocation, deps, true),
    [COMMAND.emptyTrash]: () => clearVaultTrash(deps),
  },
})
