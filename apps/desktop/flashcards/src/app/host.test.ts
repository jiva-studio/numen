import { describe, expect, it } from 'vitest'
import { createShellHost, TARGET } from './host'

const view = (kind: string, order?: number) => ({
  kind,
  component: {},
  ...(order === undefined ? {} : { placement: { pane: 'main' as const, order } }),
})

describe('createShellHost', () => {
  it('activates a plugin in the window of the flashcards target', () => {
    const { host } = createShellHost()

    expect(host.createContext('md.numen.example').target).toBe(TARGET)
  })

  it('draws the main-pane view of the lowest order, and none that has no placement', () => {
    const { host, front } = createShellHost()
    const context = host.createContext('md.numen.example')

    context.registerTabKind(view('unplaced'))
    expect(front.value).toBeNull()

    context.registerTabKind(view('second', 2))
    context.registerTabKind(view('first', 1))
    expect(front.value?.registration.kind).toBe('first')
    expect(front.value?.pluginId).toBe('md.numen.example')
  })

  it('lets go of what a plugin registered once it is disposed', () => {
    const { host, front } = createShellHost()
    host.createContext('md.numen.example').registerTabKind(view('review', 0))

    host.disposePlugin('md.numen.example')

    expect(front.value).toBeNull()
  })

  it('answers a plugin with the permissions its manifest declares', () => {
    const { host } = createShellHost()
    const context = host.createContext('md.numen.example', {
      id: 'md.numen.example',
      name: 'Example',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(context.hasPermission?.('vault:read')).toBe(true)
    expect(context.hasPermission?.('vault:write')).toBe(false)
  })
})
