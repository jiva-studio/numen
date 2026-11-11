/** When a command is available over the target the window hands the palette: its when condition. */
import { describe, expect, it } from 'vitest'
import { getPaletteValues } from './paletteValues'
import { withWhen } from './availability'
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
      'target.contentType': 'text/markdown',
      'window.isVaultOpen': true,
      'window.isVaultReady': true,
      'window.isTabOpen': true,
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
})
