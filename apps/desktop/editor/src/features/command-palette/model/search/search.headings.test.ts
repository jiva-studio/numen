/** The group headings of the search, with the real group builder and only the vault questions held back. */
import { describe, expect, it } from 'vitest'
import { CREATE_NOTE_ROW } from '../../lib/entries/entries'
import type { CommandTarget } from '../../types'
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
  creating: 'No results',
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

  it('draws only the create note row where every question returned nothing', async () => {
    const { palette, names, passages } = createHeldVault()

    palette.setTyped('Entropy').catch(() => undefined)
    await flushPromises()
    names.answer([])
    passages.answer([])
    await flushPromises()

    expect(getIds(palette)).toEqual([CREATE_NOTE_ROW])
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
})
