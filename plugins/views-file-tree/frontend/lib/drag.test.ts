/**
 * What a drag from outside the window carries, read without a screen.
 */
import { describe, expect, it } from 'vitest'
import { getDragTypes, readDroppedData } from './drag'

/** A drag holding the data given, by type. */
const createDrag = (data: Record<string, string>): DataTransfer =>
  ({ types: Object.keys(data), getData: (type: string) => data[type] ?? '' }) as DataTransfer

describe('a drag from outside the window', () => {
  it('names its types besides files', () => {
    expect(getDragTypes(createDrag({ Files: '', 'text/uri-list': 'https://numen.test/' }))).toEqual(
      ['text/uri-list'],
    )
    expect(getDragTypes(null)).toEqual([])
  })

  it('carries its data by type, without the types that carry nothing', () => {
    const drag = createDrag({
      'text/uri-list': 'https://one.test/\nhttps://two.test/',
      'text/html': '',
    })
    expect(readDroppedData(drag)).toEqual({
      'text/uri-list': 'https://one.test/\nhttps://two.test/',
    })
    expect(readDroppedData(undefined)).toEqual({})
  })
})
