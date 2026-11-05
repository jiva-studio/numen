/** When a command is available over the target the window hands it. */
import { describe, expect, it } from 'vitest'
import { isFileChosen } from './availability'
import type { CommandTarget } from '../../types'

/** A document chosen as a search result: it names a file, holds no note, and is no tab in front. */
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

describe('a command over a file that was chosen as a result', () => {
  it('is available over a document chosen from the search', () => {
    expect(isFileChosen(chosenDocument())).toBe(true)
  })

  it('is unavailable over the document shown by the tab in front', () => {
    expect(isFileChosen(chosenDocument({ tab: 'tab-1', kind: 'reader' }))).toBe(false)
  })

  it('is unavailable over a note', () => {
    expect(isFileChosen(chosenDocument({ path: 'reef/Coral.md', file: 'reef/Coral.md' }))).toBe(
      false,
    )
  })

  it('is unavailable where no vault accepts operations', () => {
    expect(isFileChosen(chosenDocument({ isReady: false }))).toBe(false)
  })

  it('is unavailable where no file is named', () => {
    expect(isFileChosen(chosenDocument({ file: '' }))).toBe(false)
  })
})
