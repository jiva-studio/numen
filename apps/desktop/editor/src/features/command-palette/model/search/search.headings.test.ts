/** The group headings of the search, with the real group builder and only the vault questions held back. */
import { describe, expect, it } from 'vitest'
import type { CommandTarget } from '@/shared/model/command'
import { createDeferred } from '@/testing/deferred'
import { useSearch, type NameMatch, type Passage, type SearchDeps, type Words } from './search'

const WORDS: Words = {
  names: 'Names',
  text: 'Text',
  meaning: 'Meaning',
  read: 'Open note',
  readAt: 'Open at this heading',
  readDocument: 'Open the document',
  noneFound: 'Nothing',
  noQuery: 'Search failed',
  wordsOnly: 'Words only',
  notEmbedded: 'Not read for meaning yet',
  creates: 'Create note',
  asChild: 'Create as child',
  asParent: 'Create as parent',
  asJump: 'Create as jump',
}

const target: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Being.md',
  title: 'Being',
  file: 'physics/Being.md',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const coral: NameMatch = {
  path: 'reef/Coral.md',
  title: 'Coral',
  heading: '',
  line: -1,
  at: [{ start: 0, end: 3 }],
  type: 'note',
}

const flushPromises = async () => {
  for (let turn = 0; turn < 6; turn += 1) await Promise.resolve()
}

function createHeldVault() {
  const names = createDeferred<readonly NameMatch[]>()
  const passages = createDeferred<readonly Passage[]>()
  const core: SearchDeps = { names: () => names.promise, search: () => passages.promise }
  const palette = useSearch(core, WORDS, { wait: () => Promise.resolve(), getTarget: () => target })
  return { palette, names, passages }
}

const getIds = (palette: ReturnType<typeof useSearch>) =>
  palette.groups.value.map((one) => one.id)

describe('the headings of the search groups', () => {
  it('draws no group while the questions are out', async () => {
    const { palette } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()

    expect(palette.groups.value).toEqual([])
  })

  it('draws no row where every question returned nothing', async () => {
    const { palette, names, passages } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()
    names.answer([])
    passages.answer([])
    await flushPromises()

    expect(palette.groups.value.flatMap((one) => one.items)).toEqual([])
  })

  it('draws the names group with its rows once names are found', async () => {
    const { palette, names } = createHeldVault()

    palette.setTyped('cor').catch(() => undefined)
    await flushPromises()
    names.answer([coral])
    await flushPromises()

    expect(getIds(palette)).toEqual(['names'])
    expect(palette.groups.value[0]?.items.map((one) => one.title)).toEqual(['Coral'])
  })

  it('draws no searching group while the questions are out', async () => {
    const { palette } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()

    expect(getIds(palette)).not.toContain('searching')
    expect(palette.status.isNothingFound.value).toBe(false)
  })

  it('is busy while a question is out', async () => {
    const { palette } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()

    expect(palette.status.isBusy.value).toBe(true)
  })

  it('is busy until the last question answers', async () => {
    const { palette, names, passages } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()
    names.answer([])
    await flushPromises()
    expect(palette.status.isBusy.value).toBe(true)

    passages.answer([])
    await flushPromises()
    expect(palette.status.isBusy.value).toBe(false)
  })

  it('is not busy before anything is typed', () => {
    const { palette } = createHeldVault()

    expect(palette.status.isBusy.value).toBe(false)
  })

  it('says nothing found once every question answered with no row', async () => {
    const { palette, names, passages } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()
    names.answer([])
    passages.answer([])
    await flushPromises()

    expect(palette.groups.value).toEqual([])
    expect(palette.status.isNothingFound.value).toBe(true)
  })

  it('draws no nothing found while a group has rows', async () => {
    const { palette, names } = createHeldVault()

    palette.setTyped('cor').catch(() => undefined)
    await flushPromises()
    names.answer([coral])
    await flushPromises()

    expect(getIds(palette)).toEqual(['names'])
    expect(palette.status.isNothingFound.value).toBe(false)
  })
})
