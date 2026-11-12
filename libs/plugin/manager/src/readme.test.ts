import { describe, expect, it } from 'vitest'
import { getReadmeText } from './readme'

describe('getReadmeText', () => {
  it('removes the leading title and keeps the body', () => {
    expect(getReadmeText('# Notes\n\nTakes notes.\n')).toBe('Takes notes.')
  })

  it('removes every HTML comment', () => {
    expect(getReadmeText('Summary.\n\n<!-- more -->\n\nDetails.<!--\nhidden\n-->')).toBe(
      'Summary.\n\n\n\nDetails.',
    )
  })

  it('removes an unclosed comment up to the end', () => {
    expect(getReadmeText('Summary.\n\n<!-- draft\nDetails.')).toBe('Summary.')
  })

  it('removes a comment that the removal of another one joins together', () => {
    expect(getReadmeText('Summary.<!<!-- inner -->-- outer -->Details.')).toBe('Summary.Details.')
  })

  it('removes a title that follows a comment', () => {
    expect(getReadmeText('<!-- generated -->\n# Notes\nBody')).toBe('Body')
  })

  it('keeps a heading of a lower level and a title further down', () => {
    expect(getReadmeText('## Setup\n\nText\n\n# Later')).toBe('## Setup\n\nText\n\n# Later')
  })

  it('keeps a hash with no space after it', () => {
    expect(getReadmeText('#tag\n\nBody')).toBe('#tag\n\nBody')
  })

  it('answers an empty text for a README that holds only a title', () => {
    expect(getReadmeText('# Only a title')).toBe('')
  })
})
