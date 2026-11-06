import { describe, expect, it } from 'vitest'
import { formatShortcut } from './shortcut'
import { formatTooltip } from './title'

describe('formatShortcut', () => {
  it('writes the window modifier as Ctrl on Linux and Windows', () => {
    expect(formatShortcut('mod+shift+e', 'linux')).toBe('Ctrl+Shift+E')
    expect(formatShortcut('mod+x', 'windows')).toBe('Ctrl+X')
  })

  it('writes the window modifier as the Command key on macOS', () => {
    expect(formatShortcut('mod+shift+e', 'darwin')).toBe('⌘+Shift+E')
  })

  it('writes Alt and a named key as the keyboard labels them', () => {
    expect(formatShortcut('ctrl+alt+enter', 'linux')).toBe('Ctrl+Alt+Enter')
  })

  it('reads the keys in any case and with spaces around them', () => {
    expect(formatShortcut('Mod + Shift + P', 'windows')).toBe('Ctrl+Shift+P')
  })

  it('writes nothing for a command without a shortcut', () => {
    expect(formatShortcut('', 'linux')).toBe('')
  })
})

describe('formatTooltip', () => {
  it('writes the title and the shortcut', () => {
    expect(formatTooltip('Open tide chart', 'Ctrl+Shift+E')).toBe('Open tide chart (Ctrl+Shift+E)')
  })

  it('writes only the title when there is no shortcut', () => {
    expect(formatTooltip('Open tide chart', '')).toBe('Open tide chart')
  })
})
