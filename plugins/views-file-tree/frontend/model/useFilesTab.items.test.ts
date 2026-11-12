/**
 * A command of the menu of a row that offers choices: a submenu of its choices, read when the menu
 * opens, and a run of the command with the choice picked.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { type CommandItem, type CommandTarget, type SlotCommand } from '@numen/plugin-sdk'
import type { Entry } from '../types'
import { useFilesTab } from './useFilesTab'
import { createFileTree, ROOT } from './useFileTree'
import { TRASH } from '../fixtures/host'
import { WORDS } from '../words'

const VERSIONS: SlotCommand = {
  kind: 'command',
  key: 'org.example.scan/command/version.delete',
  pluginId: 'org.example.scan',
  command: 'org.example.scan:version.delete',
  title: 'Delete version',
  shortcut: '',
  hasItems: true,
}

const CHOICES: readonly CommandItem[] = [
  { id: 'ocr/1', title: 'Recognised text' },
  { id: 'fix/2', title: 'Proofread text' },
]

const entry = (path: string): Entry => ({
  path,
  name: path,
  isFolder: false,
  isNote: false,
  type: 'note',
})

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const setUp = async (answer: () => Promise<readonly CommandItem[]>) => {
  const choiceRequests: [string, CommandTarget][] = []
  const runs: [string, CommandTarget, string | undefined][] = []
  const errors: string[] = []
  let release: (() => void) | null = null
  const list = createFileTree({ list: async () => [entry('Scan.pdf')] })
  const one = useFilesTab(list, {
    window: {
      getEntryIcon: () => ({}),
      openEntry: () => undefined,
      setDraggedPaths: () => undefined,
      showError: (text) => {
        errors.push(text)
      },
    },
    menu: {
      getMenuEntries: () => [VERSIONS, { kind: 'separator', key: 'separator/1' }, TRASH],
      listItems: async (command, target) => {
        choiceRequests.push([command, target])
        await new Promise<void>((done) => {
          release = done
        })
        return answer()
      },
      runCommand: async (command, target, choice) => {
        runs.push([command, target, choice])
        return ''
      },
    },
    transfer: {
      movePath: async () => undefined,
      canDropData: () => false,
      dropData: async () => [],
    },
  })
  await list.folders.openFolder(ROOT)
  one.menu.open({ path: 'Scan.pdf', at: { x: 0, y: 0 } })
  const answerChoices = async () => {
    release?.()
    await settle()
  }
  return { one, choiceRequests, runs, errors, answerChoices }
}

const getChildren = (one: Awaited<ReturnType<typeof setUp>>['one']) =>
  one.menu
    .getItems('Scan.pdf')
    .find((item) => item.id === VERSIONS.key)
    ?.children?.map((child) => ({ text: child.text, isDisabled: child.isDisabled ?? false }))

describe('a command of the menu that offers choices', () => {
  it('asks for its choices over the row when the menu opens', async () => {
    const { choiceRequests } = await setUp(async () => CHOICES)

    expect(choiceRequests).toStrictEqual([
      [
        VERSIONS.command,
        { path: 'Scan.pdf', title: 'Scan.pdf', isFolder: false, others: [], tab: '' },
      ],
    ])
  })

  it('is a submenu holding a row that says it is loading until its choices are read', async () => {
    const { one } = await setUp(async () => CHOICES)

    expect(getChildren(one)).toStrictEqual([{ text: WORDS.loadingItems, isDisabled: true }])
  })

  it('is a submenu of its choices once they are read', async () => {
    const { one, answerChoices } = await setUp(async () => CHOICES)

    await answerChoices()

    expect(getChildren(one)).toStrictEqual([
      { text: 'Recognised text', isDisabled: false },
      { text: 'Proofread text', isDisabled: false },
    ])
  })

  it('is hidden where it offers no choice', async () => {
    const { one, answerChoices } = await setUp(async () => [])

    await answerChoices()

    expect(one.menu.getItems('Scan.pdf').some((item) => item.id === VERSIONS.key)).toBe(false)
  })

  it('runs the command over the row with the id of the choice picked', async () => {
    const { one, runs, answerChoices } = await setUp(async () => CHOICES)
    await answerChoices()
    const picked = one.menu.getItems('Scan.pdf').find((item) => item.id === VERSIONS.key)!
      .children![1]!.id

    one.menu.choose(picked)

    expect(runs).toStrictEqual([
      [
        VERSIONS.command,
        { path: 'Scan.pdf', title: 'Scan.pdf', isFolder: false, others: [], tab: '' },
        'fix/2',
      ],
    ])
  })

  it('runs nothing for the loading row, and shows a failure to read the choices', async () => {
    const { one, runs, errors, answerChoices } = await setUp(async () => {
      throw new Error('the plugin is gone')
    })
    const loading = one.menu.getItems('Scan.pdf').find((item) => item.id === VERSIONS.key)!
      .children![0]!.id

    await answerChoices()
    expect(one.menu.getItems('Scan.pdf').some((item) => item.id === VERSIONS.key)).toBe(false)
    one.menu.choose(loading)

    expect(runs).toStrictEqual([])
    expect(errors).toHaveLength(1)
  })
})

describe('the menu of a row whose commands offer items', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('is not shown while the items are read, and is shown once they are', async () => {
    const { one, answerChoices } = await setUp(async () => CHOICES)

    expect(one.menu.request.value).toBeNull()
    await answerChoices()

    expect(one.menu.request.value?.path).toBe('Scan.pdf')
  })

  it('is shown with its loading row once the hold has passed', async () => {
    vi.useFakeTimers()
    const { one } = await setUp(async () => CHOICES)

    await vi.advanceTimersByTimeAsync(150)

    expect(one.menu.request.value?.path).toBe('Scan.pdf')
    expect(getChildren(one)).toStrictEqual([{ text: WORDS.loadingItems, isDisabled: true }])
  })

  it('is not shown after Escape is pressed while the items are read', async () => {
    const { one, answerChoices } = await setUp(async () => CHOICES)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await answerChoices()

    expect(one.menu.request.value).toBeNull()
  })
})
