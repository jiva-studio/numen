/**
 * The commands offered for the current selection, what a command executes on
 * it, and which commands appear on the row of another command.
 */
import { describe, expect, it } from 'vitest'
import { getGroupCommands, isCommandPrefixTyped } from './commands'
import { createAppCommands } from '@/testing/appCommands'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../../words'
import { getPrompt } from '../step/step'
import type { CommandTarget } from '@/shared/model/command'

const words = { ...windowWords, ...PALETTE_WORDS }

describe('the character that means the commands', () => {
  it('is one typed into a field holding nothing', () => {
    expect(isCommandPrefixTyped('', '>')).toBe(true)
  })

  it('is not one typed into a field holding words, so a search for one remains', () => {
    expect(isCommandPrefixTyped('foo', '>foo')).toBe(false)
    expect(isCommandPrefixTyped('', '>foo')).toBe(false)
    expect(isCommandPrefixTyped('foo', '>')).toBe(false)
    expect(isCommandPrefixTyped('>', '>>')).toBe(false)
  })
})

describe('the commands over a note', () => {
  it('leave out the ones another command reaches on its own row', () => {
    const commands = getGroupCommands(createAppCommands(), 'note').map((one) => one.id)

    expect(commands).toContain('app:file.open')
    expect(commands).not.toContain('app:file.open#beside')
  })
})

describe('the commands over the vaults an installation holds', () => {
  it('are offered in the group of the vault in front', () => {
    const commands = createAppCommands()
      .filter((one) => one.group === 'vault')
      .map((one) => one.id)

    expect(commands).toStrictEqual([
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

describe('the commands that ask for a choice among a few', () => {
  it('are the commands over a tab and a text version only, since the settings offer theirs', () => {
    const choosing = createAppCommands()
      .filter((one) => one.input === 'items')
      .map((one) => one.id)

    expect(choosing.sort()).toStrictEqual(
      ['app:tab.moveToWindow', 'app:tab.link', 'app:version.delete'].sort(),
    )
  })
})

describe('getPrompt', () => {
  const target: CommandTarget = {
    tab: 't',
    kind: 'note',
    path: 'a.md',
    title: 'A',
    file: '',
    vault: { id: 'v', name: 'V' },
    isReady: true,
  }

  it('returns the prompt at promptIndex for arguments steps and undefined otherwise', () => {
    const commandWithPrompts = {
      id: 'cmd',
      text: 'Cmd',
      group: 'note' as const,
      isAvailable: () => true,
      prompts: [
        { name: 'p1', title: 'First', isRequired: true },
        { name: 'p2', title: 'Second', isRequired: false },
      ],
    }
    expect(getPrompt({ step: 'name', command: commandWithPrompts, on: target })).toBeUndefined()
    expect(getPrompt({ step: 'arguments', command: commandWithPrompts, on: target })?.name).toBe(
      'p1',
    )
    expect(
      getPrompt({ step: 'arguments', command: commandWithPrompts, on: target, promptIndex: 1 })
        ?.name,
    ).toBe('p2')
    expect(
      getPrompt({
        step: 'arguments',
        command: { id: 'bare', text: 'Bare', group: 'note', isAvailable: () => true },
        on: target,
      }),
    ).toBeUndefined()
  })
})

describe('the command that opens a document chosen as a result', () => {
  const open = () => createAppCommands().find((one) => one.id === 'app:file.open#document')

  it('is filed under the file group with the words that open a document', () => {
    expect(open()).toMatchObject({ text: words.readDocument, group: 'file' })
  })

  it('needs no input, so it runs as soon as it is chosen', () => {
    expect(open()).toBeDefined()
    expect(open()?.input).toBeUndefined()
  })

  it('is not one of the commands over a note, so the menu of a file row does not list it', () => {
    expect(getGroupCommands(createAppCommands(), 'note').map((one) => one.id)).not.toContain(
      'app:file.open#document',
    )
  })

  it('is available over a chosen document and over nothing else', () => {
    const target: CommandTarget = {
      tab: '',
      kind: null,
      path: '',
      title: '',
      file: 'library/atlas.pdf',
      vault: { id: 'v', name: 'V' },
      isReady: true,
    }
    const runs = { canRun: () => true, cannotRun: () => undefined }

    expect(open()?.isAvailable(target, runs)).toBe(true)
    expect(open()?.isAvailable({ ...target, tab: 'tab-1', kind: 'reader' }, runs)).toBe(false)
    expect(open()?.isAvailable({ ...target, path: 'a.md', file: 'a.md' }, runs)).toBe(false)
  })
})
