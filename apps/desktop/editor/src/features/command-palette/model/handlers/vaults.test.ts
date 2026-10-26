/**
 * What a command does to the vaults an installation holds, without a window:
 * which calls it makes, and what it says when the list refuses.
 */
import { describe, expect, it, vi } from 'vitest'
import { asFailure, asValue } from '@numen/wire'
import type { Vault, VaultErrorCode, VaultResult } from '@/entities/vault'
import { createInvocation } from '../../lib/offers/invocation'
import type { CommandInvocation, CommandTarget } from '../../types'
import { ANSWER_WORDS as words } from '../../words'
import type { VaultContext, Voice } from './deps'
import { addVault, removeVault, renameVault, showVault } from './vaults'

const vault = (id: string, name: string): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: false,
})

const unused = async (): Promise<never> => {
  throw new Error('empty selection')
}

/** The vaults of an installation, answering as the test says and remembering the calls. */
const createWindow = (
  answers: {
    opened?: VaultErrorCode | null
    chose?: string
    added?: VaultResult
    renamed?: VaultResult
    removed?: VaultErrorCode | null
  } = {},
) => {
  const messages: { text: string; kind: string | undefined }[] = []
  const calls = {
    open: vi.fn<(id: string) => Promise<VaultErrorCode | null>>(),
    add: vi.fn<(path: string, name: string) => Promise<VaultResult>>(),
    rename: vi.fn<(id: string, name: string) => Promise<VaultResult>>(),
    remove: vi.fn<(id: string, trash: boolean) => Promise<VaultErrorCode | null>>(),
    reload: vi.fn<() => void>(),
    setVaultName: vi.fn<(vault: { id: string; name: string }) => void>(),
  }
  calls.open.mockResolvedValue(answers.opened ?? null)
  calls.add.mockResolvedValue(answers.added ?? asValue(vault('heat', 'Heat')))
  calls.rename.mockResolvedValue(answers.renamed ?? asValue(vault('physics', 'Heat')))
  calls.remove.mockResolvedValue(answers.removed ?? null)
  const on: VaultContext & Voice = {
    files: {
      createNote: unused,
      rename: unused,
      remove: unused,
      move: unused,
      createFolder: unused,
    },
    makers: {
      createUrl: unused,
    },
    vaults: {
      list: async () => ({ vaults: [], showing: '' }),
      choose: async () => answers.chose ?? '/vaults/Heat',
      add: calls.add,
      rename: calls.rename,
      remove: calls.remove,
      open: calls.open,
      reload: calls.reload,
      setVaultName: calls.setVaultName,
    },
    writeMessage: (text, kind) => void messages.push({ text, kind }),
  }
  return { on, calls, messages }
}

const target: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const createRenameInvocation = (name = ''): CommandInvocation =>
  createInvocation('renameVault', target, name)

describe('showing another vault', () => {
  it('opens it and draws the page again', async () => {
    const one = createWindow()

    await showVault('heat', one.on, words)

    expect(one.calls.open).toHaveBeenCalledWith('heat')
    expect(one.calls.reload).toHaveBeenCalledOnce()
  })

  it('asks for nothing where no vault is named', async () => {
    const one = createWindow()

    await showVault('', one.on, words)

    expect(one.calls.open).not.toHaveBeenCalled()
    expect(one.calls.reload).not.toHaveBeenCalled()
  })

  it('says why it would not open, as an error, and draws nothing again', async () => {
    const one = createWindow({ opened: 'showing' })

    await showVault('heat', one.on, words)

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.showing, kind: 'error' }])
    expect(one.calls.reload).not.toHaveBeenCalled()
  })
})

describe('adding a vault', () => {
  it('adds the folder chosen and opens the vault it became', async () => {
    const one = createWindow()

    await addVault(one.on, words)

    expect(one.calls.add).toHaveBeenCalledWith('/vaults/Heat', '')
    expect(one.calls.open).toHaveBeenCalledWith('heat')
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ added: asFailure('overlaps' as const) })

    await addVault(one.on, words)

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.overlaps, kind: 'error' }])
    expect(one.calls.open).not.toHaveBeenCalled()
  })

  it('opens nothing where the list is as it was', async () => {
    const one = createWindow({ added: asValue(null) })

    await addVault(one.on, words)

    expect(one.calls.open).not.toHaveBeenCalled()
    expect(one.messages).toStrictEqual([])
  })

  it('adds nothing where no folder was chosen', async () => {
    const one = createWindow({ chose: '' })

    await addVault(one.on, words)

    expect(one.calls.add).not.toHaveBeenCalled()
  })
})

describe('renaming a vault', () => {
  it('names it what was typed and notifies the window', async () => {
    const one = createWindow()

    await renameVault(createRenameInvocation('Heat'), one.on, words)

    expect(one.calls.rename).toHaveBeenCalledWith('physics', 'Heat')
    expect(one.calls.setVaultName).toHaveBeenCalledWith({ id: 'physics', name: 'Heat' })
  })

  it('notifies the window nothing where the list is as it was', async () => {
    const one = createWindow({ renamed: asValue(null) })

    await renameVault(createRenameInvocation('Heat'), one.on, words)

    expect(one.calls.setVaultName).not.toHaveBeenCalled()
    expect(one.messages).toStrictEqual([])
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ renamed: asFailure('nameTaken' as const) })

    await renameVault(createRenameInvocation('Heat'), one.on, words)

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.nameTaken, kind: 'error' }])
  })

  it('asks for nothing where the name is empty or the one it has', async () => {
    const one = createWindow()

    await renameVault(createRenameInvocation(''), one.on, words)
    await renameVault(createRenameInvocation('Physics'), one.on, words)

    expect(one.calls.rename).not.toHaveBeenCalled()
  })
})

describe('taking a vault off the list', () => {
  it('asks to erase the folder only where the command erases', async () => {
    const one = createWindow()

    await removeVault(createRenameInvocation(), one.on, { shouldErase: true, words })
    await removeVault(createRenameInvocation(), one.on, { shouldErase: false, words })

    expect(one.calls.remove.mock.calls).toStrictEqual([
      ['physics', true],
      ['physics', false],
    ])
  })

  it('says nothing where the list took it off', async () => {
    const one = createWindow()

    await removeVault(createRenameInvocation(), one.on, { shouldErase: false, words })

    expect(one.messages).toStrictEqual([])
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ removed: 'lastVault' })

    await removeVault(createRenameInvocation(), one.on, { shouldErase: false, words })

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.lastVault, kind: 'error' }])
  })
})
