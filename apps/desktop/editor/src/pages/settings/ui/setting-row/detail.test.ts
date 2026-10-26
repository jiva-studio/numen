import { describe, expect, it } from 'vitest'
import { splitDetail } from './detail'

describe('splitDetail', () => {
  it('keeps a single sentence as the summary', () => {
    expect(splitDetail('Wrap long lines to fit the editor viewport')).toStrictEqual({
      summary: 'Wrap long lines to fit the editor viewport',
      more: '',
    })
    expect(splitDetail('Path to the program.')).toStrictEqual({
      summary: 'Path to the program.',
      more: '',
    })
  })

  it('moves every sentence after the first into the rest', () => {
    expect(
      splitDetail('Path to the program. Empty finds it on PATH. Restart after.'),
    ).toStrictEqual({
      summary: 'Path to the program.',
      more: 'Empty finds it on PATH. Restart after.',
    })
  })

  it('does not split inside a number or a time', () => {
    expect(splitDetail('Sampling temperature, 0.0 to 1.0')).toStrictEqual({
      summary: 'Sampling temperature, 0.0 to 1.0',
      more: '',
    })
  })
})
