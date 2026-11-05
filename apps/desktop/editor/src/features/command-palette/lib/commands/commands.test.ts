/**
 * The commands offered for the current selection, what a command executes on
 * it, and which commands appear on the row of another command.
 */
import { describe, expect, it } from 'vitest'
import { createCommands, isCommandPrefixTyped, getNoteCommands } from './commands'
import { createPathCommands } from './pathMenu'
import { getPrompt } from '../step/step'
import type { CommandTarget } from '../../types'
import { PATH_WORDS } from '../../words'
import { WORDS as words } from '@/shared/words'

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
    const commands = getNoteCommands(createCommands(words)).map((one) => one.id)

    expect(commands).toContain('app:note.open')
    expect(commands).not.toContain('app:note.openBeside')
    expect(commands).not.toContain('app:note.delete')
  })
})

describe('the commands over the vaults an installation holds', () => {
  it('are offered in the group of the vault in front', () => {
    const commands = createCommands(words)
      .filter((one) => one.group === 'vault')
      .map((one) => one.id)

    expect(commands).toStrictEqual([
      'app:note.goTo',
      'app:vault.open',
      'app:vault.create',
      'app:vault.rename',
      'app:vault.remove',
      'app:vault.erase',
    ])
  })
})

describe('the commands that ask for a choice among a few', () => {
  it('are the ones that set how the window is drawn and how it works', () => {
    const choosing = createCommands(words)
      .filter((one) => one.needs === 'choosing')
      .map((one) => one.id)

    expect(choosing).toStrictEqual([
      'app:appearance.chooseTheme',
      'app:appearance.chooseMode',
      'app:appearance.chooseInterfaceSize',
      'app:appearance.chooseTextSize',
      'app:note.chooseTitleSync',
      'app:performance.chooseProfile',
    ])
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
    expect(getPrompt({ step: 'naming', command: commandWithPrompts, on: target })).toBeUndefined()
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

describe('createPathCommands', () => {
  it('returns only remove when several rows are selected', () => {
    expect(
      createPathCommands({ isNote: true, isFolder: false }, true, [
        { id: 'c1', text: 'C1', group: 'open' },
      ]),
    ).toStrictEqual([{ id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' }])
  })

  it('builds folder menu with new and file contributions followed by copy and remove', () => {
    const items = createPathCommands({ isNote: false, isFolder: true }, false, [
      { id: 'new-item', text: 'New Item', parentId: 'new', isDisabled: true },
      { id: 'file-item', text: 'File Item', group: 'file', isDisabled: () => true },
      { id: 'ignored-open', text: 'Open Item', group: 'open' },
    ])
    expect(items).toStrictEqual([
      { id: 'new-item', text: 'New Item', group: 'file', parentId: 'new', isDisabled: true },
      { id: 'file-item', text: 'File Item', group: 'file', isDisabled: false },
      { id: 'app:note.copyPath', text: words.copy, group: 'file' },
      { id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' },
    ])
  })

  it('builds note menu with open, copy, new, relation submenu, other contributions, and remove', () => {
    const defaultNoteItems = createPathCommands({ isNote: true, isFolder: false }, false)
    expect(defaultNoteItems).toStrictEqual([
      { id: 'app:note.open', text: words.read, group: 'open' },
      { id: 'app:note.copyPath', text: words.copy, group: 'file' },
      { id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' },
    ])

    const items = createPathCommands({ isNote: true, isFolder: false }, false, [
      { id: 'open-ext', text: 'Open Ext', group: 'open', detail: 'D1', isDisabled: false },
      { id: 'rel-parent', text: 'Rel 1', parentId: 'relation' },
      { id: 'rel-group', text: 'Rel 2', group: 'relation' },
      { id: 'new-child', text: 'New Child', parentId: 'new' },
      { id: 'other-default', text: 'Other Default' },
      { id: 'other-custom', text: 'Other Custom', group: 'custom' },
    ])
    expect(items).toStrictEqual([
      { id: 'app:note.open', text: words.read, group: 'open' },
      { id: 'open-ext', text: 'Open Ext', detail: 'D1', group: 'open', isDisabled: false },
      { id: 'app:note.copyPath', text: words.copy, group: 'file' },
      { id: 'new-child', text: 'New Child', group: 'file', parentId: 'new', isDisabled: false },
      {
        id: 'relation',
        text: PATH_WORDS.relation,
        group: 'relation',
        children: [
          { id: 'rel-parent', text: 'Rel 1', group: 'relation', isDisabled: false },
          { id: 'rel-group', text: 'Rel 2', group: 'relation', isDisabled: false },
        ],
      },
      { id: 'other-default', text: 'Other Default', group: 'file', isDisabled: false },
      { id: 'other-custom', text: 'Other Custom', group: 'custom', isDisabled: false },
      { id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' },
    ])
  })

  it('builds non-note file menu excluding relation contributions and preserving parentId', () => {
    const defaultFileItems = createPathCommands({ isNote: false, isFolder: false }, false)
    expect(defaultFileItems).toStrictEqual([
      { id: 'app:note.copyPath', text: words.copy, group: 'file' },
      { id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' },
    ])

    const items = createPathCommands({ isNote: false, isFolder: false }, false, [
      { id: 'open-f', text: 'Open File', group: 'open' },
      { id: 'rel-p', text: 'Rel P', parentId: 'relation' },
      { id: 'rel-g', text: 'Rel G', group: 'relation' },
      { id: 'sub-f', text: 'Sub File', parentId: 'new' },
      { id: 'custom-f', text: 'Custom File', group: 'extra' },
    ])
    expect(items).toStrictEqual([
      { id: 'open-f', text: 'Open File', group: 'open', isDisabled: false },
      { id: 'sub-f', text: 'Sub File', group: 'file', parentId: 'new', isDisabled: false },
      { id: 'custom-f', text: 'Custom File', group: 'extra', isDisabled: false },
      { id: 'app:note.copyPath', text: words.copy, group: 'file' },
      { id: 'app:note.trash', text: PATH_WORDS.remove, group: 'remove' },
    ])
  })
})

describe('the command that opens a document chosen as a result', () => {
  const open = () => createCommands(words).find((one) => one.id === 'app:file.open')

  it('is filed under the file group with the words that open a document', () => {
    expect(open()).toMatchObject({ text: words.readDocument, group: 'file' })
  })

  it('needs no input, so it runs as soon as it is chosen', () => {
    expect(open()).toBeDefined()
    expect(open()?.needs).toBeUndefined()
  })

  it('is not one of the commands over a note, so the menu of a file row does not list it', () => {
    expect(getNoteCommands(createCommands(words)).map((one) => one.id)).not.toContain(
      'app:file.open',
    )
    expect(
      createPathCommands({ isNote: false, isFolder: false }, false).map((one) => one.id),
    ).not.toContain('app:file.open')
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
