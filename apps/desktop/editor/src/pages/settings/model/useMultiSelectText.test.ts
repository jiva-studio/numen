import { describe, expect, it } from 'vitest'
import { useMultiSelectText, type MultiSelectTextSource } from './useMultiSelectText'

const createSource = (selected: readonly string[]): MultiSelectTextSource => ({
  choices: [
    { id: 'small', text: 'Small' },
    { id: 'large', text: 'Large' },
  ],
  selected,
  placeholder: 'Pick sizes',
  getMoreText: (count) => `+${count}`,
})

describe('the line of a multiple choice', () => {
  it('is the placeholder while nothing is chosen', () => {
    expect(useMultiSelectText(() => createSource([])).value).toBe('Pick sizes')
  })

  it('is the text of the only choice', () => {
    expect(useMultiSelectText(() => createSource(['large'])).value).toBe('Large')
  })

  it('is the id of a choice the list does not hold', () => {
    expect(useMultiSelectText(() => createSource(['huge'])).value).toBe('huge')
  })

  it('adds how many more follow the first choice', () => {
    expect(useMultiSelectText(() => createSource(['large', 'small', 'huge'])).value).toBe('Large +2')
  })
})
