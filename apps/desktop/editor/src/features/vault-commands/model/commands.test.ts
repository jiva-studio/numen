/**
 * The commands over the vaults an installation holds, carried out over the real calls of the vault
 * entity: which calls each makes, and what it reports when the list rejects the request.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { VAULTS_ERROR_CODE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
import { invoke } from '@/testing/invocation'
import { VAULT_WORDS as words } from '../words'
import { createVaultCommands } from './commands'

interface Answers {
  readonly opened?: object
  readonly chose?: string
  readonly added?: object
  readonly renamed?: object
  readonly removed?: object
  readonly undeleted?: readonly string[]
}

const restores: (() => void)[] = []

afterEach(() => {
  for (const restore of restores.splice(0)) restore()
})

const wireVault = (id: string, name: string) => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: false,
})

const createWindow = (answers: Answers = {}, withHold = false) => {
  const calls: string[] = []
  const messages: { text: string; kind: string | undefined }[] = []
  const reload = vi.fn<() => void>()
  const setVaultName = vi.fn<(vault: { id: string; name: string }) => void>()
  const hold = { pause: vi.fn(async () => undefined), resume: vi.fn() }
  restores.push(
    installCoreHandlers({
      openVault: (request: { id: string }) => {
        calls.push(`open ${request.id}`)
        return answers.opened ?? {}
      },
      chooseFolder: () => ({
        isChosen: answers.chose !== '',
        path: answers.chose ?? '/vaults/Heat',
      }),
      addVault: (request: { path: string; name: string }) => {
        calls.push(`add ${request.path}`)
        return answers.added ?? { vault: wireVault('heat', 'Heat') }
      },
      renameVault: (request: { id: string; name: string }) => {
        calls.push(`rename ${request.id} ${request.name}`)
        return answers.renamed ?? { vault: wireVault(request.id, request.name) }
      },
      removeVault: (request: { id: string; trash: boolean }) => {
        calls.push(`${request.trash ? 'erase' : 'remove'} ${request.id}`)
        return answers.removed ?? {}
      },
      emptyTrash: () => {
        calls.push('emptyTrash')
        return { undeleted: answers.undeleted ?? [] }
      },
    }),
  )
  const set = createVaultCommands(
    {
      ...(withHold ? { tabsHold: hold } : {}),
      setVaultName,
      reload,
      writeMessage: (text, kind) => {
        messages.push({ text, kind })
      },
    },
    'linux',
  )
  const carry = async (invocation: CommandInvocation): Promise<void> => {
    await set.handlers[invocation.id]?.(invocation)
  }
  return { set, calls, messages, reload, setVaultName, hold, carry }
}

const target: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const withVault = (id: string): CommandTarget => ({ ...target, vault: { id, name: 'Heat' } })

describe('the commands over the vaults', () => {
  it('are all filed in the group of the vault, and carried out but the row that goes to a note', () => {
    const { set } = createWindow()

    expect(set.commands.map((one) => one.group)).toStrictEqual(set.commands.map(() => 'vault'))
    expect(Object.keys(set.handlers).sort()).toStrictEqual(
      [
        'app:vault.open',
        'app:vault.create',
        'app:vault.rename',
        'app:vault.remove',
        'app:vault.erase',
        'app:trash.empty',
      ].sort(),
    )
  })

  it('are offered in the order of the vault group', () => {
    const { set } = createWindow()

    expect(set.commands.map((one) => one.id)).toStrictEqual([
      'app:file.open#note',
      'app:vault.open',
      'app:vault.create',
      'app:vault.rename',
      'app:vault.remove',
      'app:vault.erase',
      'app:trash.empty',
    ])
  })
})

describe('showing another vault', () => {
  it('opens it and draws the page again', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.open', withVault('heat')))

    expect(one.calls).toStrictEqual(['open heat'])
    expect(one.reload).toHaveBeenCalledOnce()
  })

  it('asks for nothing where no vault is named', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.open', withVault('')))

    expect(one.calls).toStrictEqual([])
    expect(one.reload).not.toHaveBeenCalled()
  })

  it('says why it would not open, as an error, and draws nothing again', async () => {
    const one = createWindow({ opened: { error: VAULTS_ERROR_CODE.CURRENT } })

    await one.carry(invoke('app:vault.open', withVault('heat')))

    expect(words.vaultErrors.current).toBeTypeOf('string')
    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.current, kind: 'error' }])
    expect(one.reload).not.toHaveBeenCalled()
  })

  it('holds the writes of the open tabs while the window changes vault', async () => {
    const one = createWindow({}, true)

    await one.carry(invoke('app:vault.open', withVault('heat')))

    expect(one.hold.pause).toHaveBeenCalledOnce()
    expect(one.hold.resume).not.toHaveBeenCalled()
  })

  it('lets the writes go again where the vault would not open', async () => {
    const one = createWindow({ opened: { error: VAULTS_ERROR_CODE.CURRENT } }, true)

    await one.carry(invoke('app:vault.open', withVault('heat')))

    expect(one.hold.resume).toHaveBeenCalledOnce()
  })
})

describe('adding a vault', () => {
  it('adds the folder chosen and opens the vault it became', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.create', target))

    expect(one.calls).toStrictEqual(['add /vaults/Heat', 'open heat'])
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ added: { error: VAULTS_ERROR_CODE.OVERLAPS } })

    await one.carry(invoke('app:vault.create', target))

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.overlaps, kind: 'error' }])
    expect(one.calls).toStrictEqual(['add /vaults/Heat'])
  })

  it('opens nothing where the list is as it was', async () => {
    const one = createWindow({ added: {} })

    await one.carry(invoke('app:vault.create', target))

    expect(one.calls).toStrictEqual(['add /vaults/Heat'])
    expect(one.messages).toStrictEqual([])
  })

  it('adds nothing where no folder was chosen', async () => {
    const one = createWindow({ chose: '' })

    await one.carry(invoke('app:vault.create', target))

    expect(one.calls).toStrictEqual([])
  })
})

describe('renaming a vault', () => {
  it('names it what was typed and notifies the window', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.rename', target, 'Heat'))

    expect(one.calls).toStrictEqual(['rename physics Heat'])
    expect(one.setVaultName).toHaveBeenCalledWith({ id: 'physics', name: 'Heat' })
  })

  it('notifies the window nothing where the list is as it was', async () => {
    const one = createWindow({ renamed: {} })

    await one.carry(invoke('app:vault.rename', target, 'Heat'))

    expect(one.setVaultName).not.toHaveBeenCalled()
    expect(one.messages).toStrictEqual([])
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ renamed: { error: VAULTS_ERROR_CODE.NAME_TAKEN } })

    await one.carry(invoke('app:vault.rename', target, 'Heat'))

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.nameTaken, kind: 'error' }])
  })

  it('asks for nothing where the name is empty or the one it has', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.rename', target, ''))
    await one.carry(invoke('app:vault.rename', target, 'Physics'))

    expect(one.calls).toStrictEqual([])
  })
})

describe('taking a vault off the list', () => {
  it('asks to erase the folder only where the command erases', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.erase', target))
    await one.carry(invoke('app:vault.remove', target))

    expect(one.calls).toStrictEqual(['erase physics', 'remove physics'])
  })

  it('says nothing where the list took it off', async () => {
    const one = createWindow()

    await one.carry(invoke('app:vault.remove', target))

    expect(one.messages).toStrictEqual([])
  })

  it('says why the list rejected, as an error', async () => {
    const one = createWindow({ removed: { error: VAULTS_ERROR_CODE.LAST_VAULT } })

    await one.carry(invoke('app:vault.remove', target))

    expect(one.messages).toStrictEqual([{ text: words.vaultErrors.lastVault, kind: 'error' }])
  })
})

describe('the trash emptied', () => {
  it('is deleted through the vault and says nothing when every path went', async () => {
    const one = createWindow()

    await one.carry(invoke('app:trash.empty', target))

    expect(one.calls).toStrictEqual(['emptyTrash'])
    expect(one.messages).toStrictEqual([])
  })

  it('says as one error how many paths stayed', async () => {
    const one = createWindow({ undeleted: ['.trash/a.md', '.trash/b.md'] })

    await one.carry(invoke('app:trash.empty', target))

    expect(one.messages).toStrictEqual([{ text: words.undeleted(2), kind: 'error' }])
  })
})
