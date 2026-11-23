/** When a command is available over the target the window hands the palette: its when condition. */
import { describe, expect, it, vi } from 'vitest'
import { getPaletteValues } from './paletteValues'
import { withWhen } from './availability'
import { flushTab, type Notes } from './notes'
import { getValues, withValues } from './values'
import { WORDS } from '@/shared/ui/words'
import type { CommandTarget } from './invocation'

const chosenDocument = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: '',
  kind: null,
  path: '',
  title: 'Atlas',
  file: 'library/atlas.pdf',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

const noteInFront = (override: Partial<CommandTarget> = {}): CommandTarget =>
  chosenDocument({
    tab: 'tab-1',
    kind: 'note',
    path: 'reef/Coral.md',
    file: 'reef/Coral.md',
    ...override,
  })

describe('the values of the palette', () => {
  it('make the note in front the target, and say what the window holds', () => {
    expect(getPaletteValues(noteInFront(), { md: 'text/markdown' })).toMatchObject({
      'target.isNote': true,
      'target.isMultiple': false,
      'target.contentType': 'text/markdown',
      'window.isVaultOpen': true,
      'window.isVaultReady': true,
      'window.isTabOpen': true,
    })
    expect(
      getPaletteValues(noteInFront({ others: ['reef/Atoll.md'] }), { md: 'text/markdown' }),
    ).toMatchObject({
      'target.isMultiple': true,
    })
  })

  it('make the file in front the target where it is no note', () => {
    expect(getPaletteValues(chosenDocument(), { pdf: 'application/pdf' })).toMatchObject({
      'target.isNote': false,
      'target.isFile': true,
      'target.contentType': 'application/pdf',
      'window.isTabOpen': false,
    })
  })

  it('give no target while the vault does not accept operations', () => {
    expect(getPaletteValues(noteInFront({ isReady: false }), {})).toMatchObject({
      'target.isNote': false,
      'window.isVaultOpen': true,
      'window.isVaultReady': false,
    })
  })

  it('carry the settings as config', () => {
    expect(
      getPaletteValues(noteInFront(), {}, { editor: { lineNumbers: true } }).config,
    ).toStrictEqual({
      editor: { lineNumbers: true },
    })
  })
})

describe('a command over a file that was chosen as a result', () => {
  const isAvailable = withWhen('target.isFile && !window.isTabOpen').isAvailable

  it('is available over a document chosen from the search', () => {
    expect(isAvailable(chosenDocument())).toBe(true)
  })

  it('is unavailable over the document shown by the tab in front', () => {
    expect(isAvailable(chosenDocument({ tab: 'tab-1', kind: 'reader' }))).toBe(false)
  })

  it('is unavailable over a note', () => {
    expect(isAvailable(chosenDocument({ path: 'reef/Coral.md', file: 'reef/Coral.md' }))).toBe(
      false,
    )
  })

  it('is unavailable where no vault accepts operations', () => {
    expect(isAvailable(chosenDocument({ isReady: false }))).toBe(false)
  })

  it('is unavailable where no file is named', () => {
    expect(isAvailable(chosenDocument({ file: '' }))).toBe(false)
  })
})

describe('a command without a condition', () => {
  it('is always available', () => {
    expect(withWhen('').isAvailable(chosenDocument({ isReady: false, file: '' }))).toBe(true)
  })

  it('rejects a malformed condition', () => {
    expect(() => withWhen('target.isFile &&')).toThrow()
  })
})

describe('the values and tab flush of a command', () => {
  it('omits empty values and keeps non-empty values', () => {
    const base = {
      id: 'app:file.open',
      target: { ...noteInFront(), others: [] },
      note: 'Entropy.md',
      name: '',
    }
    expect(getValues(undefined)).toStrictEqual({})
    expect(getValues({})).toStrictEqual({})
    expect(getValues({ placement: 'split' })).toStrictEqual({ values: { placement: 'split' } })
    expect(withValues(base, {})).toStrictEqual(base)
    expect(withValues(base, { placement: 'split' })).toStrictEqual({
      ...base,
      values: { placement: 'split' },
    })
  })

  it('reports when no tab matches a path and formats undeleted counts', async () => {
    const notes: Notes = {
      getTabAt: () => null,
      getPath: (id) => id,
      isAsking: () => false,
      flushPending: vi.fn(() => Promise.resolve()),
      close: () => undefined,
      openFile: () => undefined,
    }
    expect(await flushTab('reef/Coral.md', notes)).toStrictEqual({
      tabId: null,
      isWaiting: false,
    })
    expect(WORDS.undeleted(1)).toBe('Not deleted: 1 item in the trash')
    expect(WORDS.undeleted(2)).toBe('Not deleted: 2 items in the trash')
  })
})
