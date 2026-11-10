import { describe, expect, it, vi } from 'vitest'
import { createWhenReader } from '../when'
import { createKeybindingRegistry, type KeybindingSource } from './keybindingRegistry'

const press = (init: KeyboardEventInit) => new KeyboardEvent('keydown', init)
const CTRL_P = press({ code: 'KeyP', ctrlKey: true })

function create(isMac = false) {
  const report = vi.fn()
  const registry = createKeybindingRegistry({ isMac: () => isMac, whenReader: createWhenReader(), report })
  return { registry, report }
}

const bind = (key: string, command: string, extra: Partial<KeybindingSource> = {}) => ({
  key,
  command,
  ...extra,
})

describe('the keybinding registry', () => {
  it('matches the event of a key to the entry that holds it, with its owner and the full command id', () => {
    const { registry } = create()
    registry.register('org.example.ledger', [bind('mod+p', 'ledger.open', { values: { a: 'b' } })])

    expect(registry.match(CTRL_P, {})).toMatchObject({
      owner: 'org.example.ledger',
      key: 'mod+p',
      command: 'org.example.ledger:ledger.open',
      values: { a: 'b' },
    })
  })

  it('keeps the full id of a command of another owner', () => {
    const { registry } = create()
    registry.register('org.example.ledger', [bind('mod+p', 'app:commands.open')])

    expect(registry.match(CTRL_P, {})?.command).toBe('app:commands.open')
  })

  it('matches nothing for another key, and for no keybindings', () => {
    const { registry } = create()

    expect(registry.match(CTRL_P, {})).toBeUndefined()

    registry.register('app', [bind('mod+k', 'app:vault.search')])

    expect(registry.match(CTRL_P, {})).toBeUndefined()
  })

  it('reads mod as Meta on macOS and as Ctrl elsewhere', () => {
    const mac = create(true).registry
    const other = create(false).registry
    mac.register('app', [bind('mod+p', 'app:commands.open')])
    other.register('app', [bind('mod+p', 'app:commands.open')])

    expect(mac.match(press({ code: 'KeyP', metaKey: true }), {})).toBeDefined()
    expect(mac.match(CTRL_P, {})).toBeUndefined()
    expect(other.match(CTRL_P, {})).toBeDefined()
    expect(other.match(press({ code: 'KeyP', metaKey: true }), {})).toBeUndefined()
  })

  it('reads mod as Meta, not Ctrl, when the platform turns out to be macOS after the registration', () => {
    let isMac = false
    const registry = createKeybindingRegistry({
      isMac: () => isMac,
      whenReader: createWhenReader(),
    })
    registry.register('app', [bind('mod+p', 'app:commands.open')])
    expect(registry.match(CTRL_P, {})).toBeDefined()

    isMac = true

    expect(registry.match(press({ code: 'KeyP', metaKey: true }), {})).toBeDefined()
    expect(registry.match(CTRL_P, {})).toBeUndefined()
  })

  it('matches the Russian layout by the physical key', () => {
    const { registry } = create()
    registry.register('app', [bind('mod+p', 'app:commands.open')])

    expect(registry.match(press({ code: 'KeyP', key: 'з', ctrlKey: true }), {})).toBeDefined()
  })

  it('does not change anything when it matches', () => {
    const { registry } = create()
    registry.register('app', [bind('mod+p', 'app:commands.open')])
    const event = press({ code: 'KeyP', ctrlKey: true, cancelable: true })

    registry.match(event, {})

    expect(event.defaultPrevented).toBe(false)
  })

  describe('conditions', () => {
    it('match only where the condition holds over the context', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+p', 'app:commands.open', { when: '!window.isFieldFocused' })])

      expect(registry.match(CTRL_P, { 'window.isFieldFocused': false })).toBeDefined()
      expect(registry.match(CTRL_P, { 'window.isFieldFocused': true })).toBeUndefined()
    })

    it('treat an empty condition as always true', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+p', 'app:commands.open', { when: '' })])

      expect(registry.match(CTRL_P, {})).toBeDefined()
    })

    it('leave out a keybinding whose condition cannot be read, and report it once', () => {
      const { registry, report } = create()
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open', { when: 'nowhere.at.all' })])
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open', { when: 'nowhere.at.all' })])

      expect(registry.match(CTRL_P, {})).toBeUndefined()
      expect(report).toHaveBeenCalledOnce()
      expect(report).toHaveBeenCalledWith('org.example.ledger', expect.stringContaining('nowhere.at.all'))
    })

    it('leave out a keybinding whose key cannot be read, and report it', () => {
      const { registry, report } = create()
      registry.register('org.example.ledger', [bind('hyper+p', 'ledger.open')])

      expect(registry.match(CTRL_P, {})).toBeUndefined()
      expect(report).toHaveBeenCalledWith('org.example.ledger', expect.stringContaining('hyper+p'))
    })

    it('leave out a keybinding that canRun refuses, so a weaker one can win', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+p', 'app:commands.open')])
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open', { when: '!window.isPaletteOpen' })])

      const found = registry.match(CTRL_P, {}, (one) => one.owner !== 'app')

      expect(found?.command).toBe('org.example.ledger:ledger.open')
    })
  })

  describe('a conflict of two keybindings of one key', () => {
    it('is won by the one whose condition reads more variables, with no report', () => {
      const { registry, report } = create()
      registry.register('app', [bind('mod+p', 'app:commands.open')])
      registry.register('org.example.ledger', [
        bind('mod+p', 'ledger.open', { when: 'window.isFieldFocused' }),
      ])

      expect(registry.match(CTRL_P, { 'window.isFieldFocused': true })?.command).toBe(
        'org.example.ledger:ledger.open',
      )
      expect(registry.match(CTRL_P, { 'window.isFieldFocused': false })?.command).toBe(
        'app:commands.open',
      )
      expect(report).not.toHaveBeenCalled()
    })

    it('counts the variables a condition reads, not its text', () => {
      const { registry } = create()
      registry.register('org.example.a', [
        bind('mod+p', 'a.open', { when: 'window.isFieldFocused' }),
      ])
      registry.register('org.example.b', [
        bind('mod+p', 'b.open', { when: 'window.isFieldFocused && !window.isPaletteOpen' }),
      ])

      expect(registry.match(CTRL_P, { 'window.isFieldFocused': true })?.command).toBe('org.example.b:b.open')
    })

    it('is won by the application over a plugin when the conditions are as wide', () => {
      const { registry, report } = create()
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open')])
      registry.register('app', [bind('mod+p', 'app:commands.open')])

      expect(registry.match(CTRL_P, {})?.command).toBe('app:commands.open')
      expect(report).toHaveBeenCalledOnce()
      expect(report).toHaveBeenCalledWith('app', expect.stringContaining('mod+p'))
    })

    it('is won by the one registered first between two plugins, and reported once however often it is matched', () => {
      const { registry, report } = create()
      registry.register('org.example.a', [bind('mod+p', 'a.open')])
      registry.register('org.example.b', [bind('mod+p', 'b.open')])

      expect(registry.match(CTRL_P, {})?.command).toBe('org.example.a:a.open')
      registry.match(CTRL_P, {})

      expect(report).toHaveBeenCalledOnce()
    })

    it('is no conflict when the other condition does not hold', () => {
      const { registry, report } = create()
      registry.register('org.example.a', [bind('mod+p', 'a.open', { when: 'window.isPaletteOpen' })])
      registry.register('org.example.b', [bind('mod+p', 'b.open')])

      registry.match(CTRL_P, { 'window.isPaletteOpen': false })

      expect(report).not.toHaveBeenCalled()
    })

    it('is no conflict between two keybindings that run the same command with the same values', () => {
      const { registry, report } = create()
      registry.register('app', [bind('mod+p', 'app:commands.open')])
      registry.register('org.example.a', [bind('mod+p', 'app:commands.open')])

      registry.match(CTRL_P, {})

      expect(report).not.toHaveBeenCalled()
    })
  })

  describe('unregistering', () => {
    it('removes the keybindings of an owner and leaves those of the others', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+k', 'app:vault.search')])
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open')])

      registry.unregister('org.example.ledger')

      expect(registry.match(CTRL_P, {})).toBeUndefined()
      expect(registry.match(press({ code: 'KeyK', ctrlKey: true }), {})).toBeDefined()
    })

    it('is undone by registering the owner again, which replaces what it registered before', () => {
      const { registry } = create()
      registry.register('org.example.ledger', [bind('mod+p', 'ledger.open')])
      registry.register('org.example.ledger', [bind('mod+k', 'ledger.find')])

      expect(registry.match(CTRL_P, {})).toBeUndefined()
      expect(registry.match(press({ code: 'KeyK', ctrlKey: true }), {})?.command).toBe(
        'org.example.ledger:ledger.find',
      )
    })

    it('does nothing for an owner that registered nothing', () => {
      const { registry } = create()

      expect(() => registry.unregister('nobody')).not.toThrow()
    })
  })

  describe('the key of a command', () => {
    it('is the key as declared, for the command and the values given', () => {
      const { registry } = create()
      registry.register('app', [
        bind('mod+g', 'app:file.open'),
        bind('mod+shift+g', 'app:file.open', { values: { placement: 'split' } }),
      ])

      expect(registry.getKeyFor('app:file.open')).toBe('mod+g')
      expect(registry.getKeyFor('app:file.open', { placement: 'split' })).toBe('mod+shift+g')
      expect(registry.getKeyFor('app:file.open', { placement: 'current' })).toBeUndefined()
    })

    it('is undefined for a command with no key, and after the owner unregisters', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+n', 'app:note.create')])

      expect(registry.getKeyFor('app:vault.search')).toBeUndefined()

      registry.unregister('app')

      expect(registry.getKeyFor('app:note.create')).toBeUndefined()
    })

    it('does not depend on the order of the names of the values', () => {
      const { registry } = create()
      registry.register('app', [bind('mod+g', 'app:file.open', { values: { b: '2', a: '1' } })])

      expect(registry.getKeyFor('app:file.open', { a: '1', b: '2' })).toBe('mod+g')
    })
  })
})
