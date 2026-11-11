/**
 * The pick list of a command with items in the palette, and the commands the palette leaves out
 * because they list no item over what is in front.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { CommandItem } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandTarget, RunSupport } from '@/shared/model/command'
import { createItemLists } from './pluginItems'

const TARGET = {
  tab: '',
  kind: null,
  path: 'tides.md',
  file: 'tides.md',
  title: 'tides',
  vault: { id: 'v', name: 'Harbor' },
  isReady: true,
} as CommandTarget

type WindowLists = Parameters<typeof createItemLists>[1]['paletteLists']

const WINDOW_LISTS: WindowLists = { getStepGroups: () => [], previewItem: () => undefined }
const RUNS: RunSupport = { canRun: () => true, cannotRun: () => undefined }

const createCommand = (id: string): Command => ({
  id,
  text: id,
  group: 'note',
  isAvailable: () => true,
})

const settle = async () => {
  for (let round = 0; round < 4; round += 1) await nextTick()
}

function setUp(itemsById: Readonly<Record<string, readonly CommandItem[]>>) {
  const listCommandItems = vi.fn(async (id: string) => itemsById[id] ?? [])
  const host = { hasCommandItems: (id: string) => id in itemsById, listCommandItems }
  const lists = createItemLists(host, {
    paletteLists: WINDOW_LISTS,
    getTarget: () => TARGET,
    runs: RUNS,
  })
  const isOpen = ref(true)
  lists.resetWhenClosed(isOpen)
  return { ...lists, listCommandItems, isOpen }
}

describe('the commands with items in the palette', () => {
  it('are left out once their items are read and none is listed', async () => {
    const { removeEmptyCommands } = setUp({
      'tab.move': [{ id: 'w1', title: 'Window 1' }],
      'version.delete': [],
    })
    const palette = [createCommand('tab.move'), createCommand('version.delete'), createCommand('x')]

    removeEmptyCommands(palette)
    await settle()

    expect(removeEmptyCommands(palette).map((one) => one.id)).toStrictEqual(['tab.move', 'x'])
  })

  it('are listed while their items are being read', () => {
    const { removeEmptyCommands } = setUp({ 'version.delete': [] })

    expect(
      removeEmptyCommands([createCommand('version.delete')]).map((one) => one.id),
    ).toStrictEqual(['version.delete'])
  })

  it('read no items of a command that is not available over what is in front', () => {
    const { removeEmptyCommands, listCommandItems } = setUp({ 'version.delete': [] })

    removeEmptyCommands([{ ...createCommand('version.delete'), isAvailable: () => false }])

    expect(listCommandItems).not.toHaveBeenCalled()
  })

  it('are read again each time the palette opens', async () => {
    const { removeEmptyCommands, isOpen, listCommandItems } = setUp({ 'version.delete': [] })
    removeEmptyCommands([createCommand('version.delete')])
    await settle()

    isOpen.value = false
    await nextTick()
    isOpen.value = true
    await nextTick()
    removeEmptyCommands([createCommand('version.delete')])

    expect(listCommandItems).toHaveBeenCalledTimes(2)
  })

  it('drop an answer read for an earlier open of the palette', async () => {
    const answers: ((items: readonly CommandItem[]) => void)[] = []
    const listCommandItems = vi.fn(
      () => new Promise<readonly CommandItem[]>((resolve) => answers.push(resolve)),
    )
    const lists = createItemLists(
      { hasCommandItems: (id: string) => id === 'zzStale', listCommandItems },
      { paletteLists: WINDOW_LISTS, getTarget: () => TARGET, runs: RUNS },
    )
    const isOpen = ref(true)
    lists.resetWhenClosed(isOpen)
    const command = [createCommand('zzStale')]
    lists.removeEmptyCommands(command)
    isOpen.value = false
    await nextTick()
    isOpen.value = true
    await nextTick()
    lists.removeEmptyCommands(command)

    answers[0]?.([])
    await settle()
    expect(lists.removeEmptyCommands(command).map((one) => one.id)).toStrictEqual(['zzStale'])
    answers[1]?.([{ id: 'w1', title: 'Window 1' }])
    await settle()

    expect(lists.lists.getStepGroups('zzStale', '')[0]?.items).toStrictEqual([
      { id: 'w1', title: 'Window 1' },
    ])
  })

  it('are read again when the target changes while the palette is open', async () => {
    const front = ref({ ...TARGET, path: 'assets', file: 'assets', isFolder: true })
    const listCommandItems = vi.fn(async (_id: string, target: { path: string }) =>
      target.path.endsWith('.pdf') ? [{ id: 'pdf', title: 'Extract text layer' }] : [],
    )
    const lists = createItemLists(
      { hasCommandItems: (id: string) => id === 'app:file.createText', listCommandItems },
      { paletteLists: WINDOW_LISTS, getTarget: () => front.value, runs: RUNS },
    )
    const isOpen = ref(true)
    lists.resetWhenClosed(isOpen)
    const command = [createCommand('app:file.createText')]
    lists.removeEmptyCommands(command)
    await settle()
    expect(lists.removeEmptyCommands(command)).toHaveLength(0)

    front.value = { ...TARGET, path: 'assets/scan.pdf', file: 'assets/scan.pdf', isFolder: false }
    lists.removeEmptyCommands(command)
    await settle()

    expect(lists.removeEmptyCommands(command).map((one) => one.id)).toStrictEqual([
      'app:file.createText',
    ])
  })

  it('are all listed while the palette is closed, and no item is read', () => {
    const { removeEmptyCommands, isOpen, listCommandItems } = setUp({ 'version.delete': [] })
    isOpen.value = false

    expect(removeEmptyCommands([createCommand('version.delete')])).toHaveLength(1)
    expect(listCommandItems).not.toHaveBeenCalled()
  })
})

describe('a shared command in the palette', () => {
  const CREATE_TEXT = 'app:file.createText'
  const createShared = (): Command => ({
    ...createCommand(CREATE_TEXT),
    text: 'Create text',
    input: 'items',
  })

  it('is listed by the title of its one handler over what is in front, and runs it with no pick', async () => {
    const { removeEmptyCommands } = setUp({
      [CREATE_TEXT]: [{ id: 'md.numen.engines.transcription', title: 'Transcribe' }],
    })

    removeEmptyCommands([createShared()])
    await settle()
    const [listed] = removeEmptyCommands([createShared()])

    expect(listed?.text).toBe('Transcribe')
    expect(listed?.input).toBeUndefined()
  })

  it('keeps its own title and its pick list where several handlers hold', async () => {
    const { removeEmptyCommands } = setUp({
      [CREATE_TEXT]: [
        { id: 'md.numen.engines.ocr', title: 'Recognize text' },
        { id: 'md.numen.readers.pdf', title: 'Extract text layer' },
      ],
    })

    removeEmptyCommands([createShared()])
    await settle()
    const [listed] = removeEmptyCommands([createShared()])

    expect(listed?.text).toBe('Create text')
    expect(listed?.input).toBe('items')
  })

  it('keeps its own title where its one item is an item of a handler, in the group of its plugin', async () => {
    const { removeEmptyCommands } = setUp({
      'app:link.downloadMedia': [
        {
          id: 'md.numen.engines.transcription:t1',
          title: 'transcript',
          group: 'md.numen.engines.transcription',
        },
      ],
    })
    const shared = {
      ...createCommand('app:link.downloadMedia'),
      text: 'Delete text',
      input: 'items' as const,
    }

    removeEmptyCommands([shared])
    await settle()

    expect(removeEmptyCommands([shared])[0]?.text).toBe('Delete text')
  })

  it('that asks for its arguments keeps its steps where one handler holds', async () => {
    const IMPORT = 'app:link.import'
    const { removeEmptyCommands } = setUp({
      [IMPORT]: [{ id: 'md.numen.tools.media.importer', title: 'Import link' }],
    })
    const shared: Command = {
      ...createCommand(IMPORT),
      text: 'Import link',
      input: 'arguments',
      prompts: [{ name: 'address', title: 'Address', isRequired: true }],
    }

    removeEmptyCommands([shared])
    await settle()

    expect(removeEmptyCommands([shared])[0]?.input).toBe('arguments')
  })

  it('is the only kind of command a single item renames', async () => {
    const { removeEmptyCommands } = setUp({ 'tab.move': [{ id: 'w1', title: 'Window 1' }] })

    removeEmptyCommands([createCommand('tab.move')])
    await settle()

    expect(removeEmptyCommands([createCommand('tab.move')])[0]?.text).toBe('tab.move')
  })
})
