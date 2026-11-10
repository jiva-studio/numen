/** The keys of a window through the one handler, over the real registry, palette model and runner. All names are invented. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createRealWindow, windowValues, type ManifestKeybinding } from '@/testing/realCommands'

const reveal = (key: string, kind: string, when?: string) => ({
  key,
  command: 'app:file.reveal',
  values: { kind },
  ...(when ? { when } : {}),
})

const create = (keybindings: readonly ManifestKeybinding[]) => {
  const navigateTab = vi.fn(async (_kind: string, _path: string) => undefined)
  const manifest = { id: 'org.example.ledger', contributes: { keybindings } }
  return { navigateTab, ...createRealWindow({ manifests: [manifest], navigateTab }) }
}

const press = (code: string, init: KeyboardEventInit = {}) =>
  new KeyboardEvent('keydown', { code, cancelable: true, ...init })

beforeEach(() => {
  windowValues['window.isFieldFocused'] = false
  windowValues['window.isPaletteOpen'] = false
})

describe('a letter typed into a field', () => {
  it('fires a bare key whose condition does not name the field', async () => {
    const one = create([reveal('x', 'plex')])
    windowValues['window.isFieldFocused'] = true
    const event = press('KeyX')

    one.used.onKeyDown(event)

    await vi.waitFor(() =>
      expect(one.navigateTab).toHaveBeenCalledExactlyOnceWith('plex', 'Tides.md'),
    )
    expect(event.defaultPrevented).toBe(true)
  })

  it('fires no bare key that guards itself with window.isFieldFocused being false', async () => {
    const one = create([reveal('x', 'plex', '!window.isFieldFocused')])
    windowValues['window.isFieldFocused'] = true
    const inField = press('KeyX')

    one.used.onKeyDown(inField)

    expect(one.navigateTab).not.toHaveBeenCalled()
    expect(inField.defaultPrevented).toBe(false)

    windowValues['window.isFieldFocused'] = false
    const free = press('KeyX')
    one.used.onKeyDown(free)

    await vi.waitFor(() => expect(one.navigateTab).toHaveBeenCalledOnce())
    expect(free.defaultPrevented).toBe(true)
  })
})

describe('a key while a palette is open', () => {
  it('fires no key that guards itself with window.isPaletteOpen being false', () => {
    const one = create([reveal('x', 'plex', '!window.isPaletteOpen')])
    windowValues['window.isPaletteOpen'] = true
    const event = press('KeyX')

    one.used.onKeyDown(event)

    expect(one.navigateTab).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })

  it('still fires a key of the application, which names no palette', () => {
    const one = create([])
    windowValues['window.isPaletteOpen'] = true

    one.used.onKeyDown(press('KeyK', { ctrlKey: true }))

    expect(one.used.palette.open.value).toBe(true)
  })
})

describe('the default of a key press', () => {
  it('is prevented only where a keybinding matched', async () => {
    const one = create([reveal('x', 'files')])
    const matched = press('KeyX')
    const unmatched = press('KeyY')

    one.used.onKeyDown(matched)
    one.used.onKeyDown(unmatched)

    expect(matched.defaultPrevented).toBe(true)
    expect(unmatched.defaultPrevented).toBe(false)
    await vi.waitFor(() =>
      expect(one.navigateTab).toHaveBeenCalledExactlyOnceWith('files', 'Tides.md'),
    )
  })

  it('is prevented for the keys of the application, which come from commands.json', () => {
    const one = create([])
    const search = press('KeyK', { ctrlKey: true })
    const shifted = press('KeyK', { ctrlKey: true, shiftKey: true })

    one.used.onKeyDown(search)
    one.used.onKeyDown(shifted)

    expect(search.defaultPrevented).toBe(true)
    expect(shifted.defaultPrevented).toBe(false)
    expect(one.used.palette.open.value).toBe(true)
  })

  it('opens the commands for its key and closes them with the same key', () => {
    const one = create([])

    one.used.onKeyDown(press('KeyP', { ctrlKey: true }))
    expect(one.used.commands.palette.open.value).toBe(true)

    one.used.onKeyDown(press('KeyP', { ctrlKey: true }))
    expect(one.used.commands.palette.open.value).toBe(false)
  })
})
