/**
 * What a drag from outside the window carries, read without a screen.
 *
 * A browser writes the list its own way: blank lines, comments, and the
 * addresses among them.
 */
import { describe, expect, it } from 'vitest'
import { addressDropped, addressIn, carriesAddress } from './drag'

/** A drag holding what a browser offers, and nothing under any other type. */
const createDrag = (written: Record<string, string>): DataTransfer =>
  ({ getData: (type: string) => written[type] ?? '' }) as DataTransfer

describe('a drag carrying an address', () => {
  it('is one the tree takes', () => {
    expect(carriesAddress(['text/uri-list', 'text/plain'])).toBe(true)
  })

  it('is not a drag of files, and not a drag of nothing', () => {
    expect(carriesAddress(['Files'])).toBe(false)
    expect(carriesAddress(undefined)).toBe(false)
  })
})

describe('the address a list holds', () => {
  it('is the line, whatever room is left around it', () => {
    expect(addressIn('  https://numen.md/  \n')).toBe('https://numen.md/')
  })

  it('is the first address, past the comments and the blank lines', () => {
    expect(addressIn('# a comment\n\nhttps://numen.md/\nhttps://numen.md/other')).toBe(
      'https://numen.md/',
    )
  })

  it('is nothing where the list holds none', () => {
    expect(addressIn('# a comment\n\n')).toBe('')
  })
})

describe('the address let go over the tree', () => {
  it('is read out of what the drag holds', () => {
    expect(addressDropped(createDrag({ 'text/uri-list': 'https://numen.md/' }))).toBe(
      'https://numen.md/',
    )
  })

  it('is nothing where the drag holds no list, and nothing where there is no drag', () => {
    expect(addressDropped(createDrag({ 'text/plain': 'https://numen.md/' }))).toBe('')
    expect(addressDropped(null)).toBe('')
  })
})
