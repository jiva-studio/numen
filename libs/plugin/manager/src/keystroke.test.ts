import { describe, expect, it } from 'vitest'
import { parseKeystroke } from './keystroke'

describe('parseKeystroke', () => {
  it('reads mod as Command on an Apple system', () => {
    expect(parseKeystroke('mod+shift+a', true)).toStrictEqual({
      icons: ['command', 'shift'],
      letter: 'A',
    })
  })

  it('reads mod as Control elsewhere', () => {
    expect(parseKeystroke('mod+k', false)).toStrictEqual({ icons: ['control'], letter: 'K' })
  })

  it('reads ctrl and alt the same on every system', () => {
    expect(parseKeystroke('ctrl+alt+p', true)).toStrictEqual({
      icons: ['control', 'option'],
      letter: 'P',
    })
  })

  it('leaves out a modifier it does not know', () => {
    expect(parseKeystroke('hyper+x', false)).toStrictEqual({ icons: [], letter: 'X' })
  })

  it('reads a key with no modifier', () => {
    expect(parseKeystroke('f2', false)).toStrictEqual({ icons: [], letter: 'F2' })
    expect(parseKeystroke('shift', false)).toStrictEqual({ icons: [], letter: 'SHIFT' })
  })
})
