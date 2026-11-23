import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, onUnmounted } from 'vue'
import { CONVERSATION } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import type { Manifest } from '@numen/core-client'
import { createPluginStyles } from './styles'
import { createViewRegistry, type MountAccess } from './views'

const NO_ACCESS: MountAccess = { inherits: false, permissions: [], requirePerm: () => undefined }

function createView(name: string, options?: { exposeFocus?: boolean; onUnmount?: () => void }) {
  const focused = { count: 0 }
  const component = defineComponent({
    props: { context: { type: Object, required: true } },
    setup(props, { expose }) {
      if (options?.exposeFocus !== false) {
        expose({ focus: () => focused.count++ })
      }
      if (options?.onUnmount) {
        onUnmounted(options.onUnmount)
      }
      return () => h('p', `${name}: ${(props.context as { focus: string }).focus}`)
    },
  })
  return { component, focused }
}

describe('createViewRegistry', () => {
  const styles = createPluginStyles()

  it('draws the declared view into the element with the context passed', () => {
    const views = createViewRegistry({ styles })
    const unmounted = vi.fn()
    const { component, focused } = createView('chat', { onUnmount: unmounted })
    views.registerView('chat.plugin', { capability: CONVERSATION, component })
    const element = document.createElement('section')

    const mounted = views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' }, NO_ACCESS)
    mounted.focus()

    expect(element.querySelector('.plugin-view')).not.toBeNull()
    expect(element.querySelector('link')).toBeNull()
    expect(element.textContent).toBe('chat: Decks/Physics.md')
    expect(focused.count).toBe(1)
    mounted.dispose()
    expect(unmounted).toHaveBeenCalledTimes(1)
    expect(element.childElementCount).toBe(0)
    mounted.dispose()
  })

  it('leaves the element empty while no plugin declares the capability, and draws once one does', () => {
    const views = createViewRegistry({ styles })
    const element = document.createElement('section')
    const disposedBeforeRegister = document.createElement('section')

    const earlyDisposed = views.mountView(
      CONVERSATION,
      disposedBeforeRegister,
      { focus: 'gone' },
      NO_ACCESS,
    )
    earlyDisposed.dispose()

    const mounted = views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' }, NO_ACCESS)
    mounted.focus()
    expect(element.childElementCount).toBe(0)

    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat', { exposeFocus: false }).component,
    })
    mounted.focus()
    expect(element.textContent).toBe('chat: Decks/Physics.md')
    expect(disposedBeforeRegister.childElementCount).toBe(0)
  })

  it('takes the view away with the plugin that declared it', () => {
    const views = createViewRegistry({ styles })
    const declared = views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const otherUnmounted = vi.fn()
    const other = views.registerView('other.plugin', {
      capability: 'graph',
      component: createView('graph', { onUnmount: otherUnmounted }).component,
    })
    const element = document.createElement('section')
    const pendingEl = document.createElement('section')
    const graphEl = document.createElement('section')
    views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' }, NO_ACCESS)
    views.mountView('unregistered', pendingEl, { focus: 'none' }, NO_ACCESS)
    views.mountView('graph', graphEl, { focus: 'Nodes' }, NO_ACCESS)

    declared.dispose()
    declared.dispose()

    const graphEl2 = document.createElement('section')
    views.mountView('graph', graphEl2, { focus: 'Nodes2' }, NO_ACCESS)
    expect(graphEl2.textContent).toBe('graph: Nodes2')

    expect(element.childElementCount).toBe(0)
    expect(otherUnmounted).not.toHaveBeenCalled()
    expect(graphEl.textContent).toBe('graph: Nodes')
    other.dispose()
    expect(otherUnmounted).toHaveBeenCalledTimes(2)
  })

  it('refuses a second plugin the capability another plugin draws, and keeps the first view', () => {
    const views = createViewRegistry({ styles })
    views.registerView('first.plugin', {
      capability: CONVERSATION,
      component: createView('first').component,
    })
    const element = document.createElement('section')
    views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' }, NO_ACCESS)

    expect(() =>
      views.registerView('second.plugin', {
        capability: CONVERSATION,
        component: createView('second').component,
      }),
    ).toThrow(/"first.plugin" already draws/)

    expect(element.textContent).toBe('first: Decks/Physics.md')
  })

  it('refuses the same plugin to register a capability twice', () => {
    const views = createViewRegistry({ styles })
    const registration = { capability: CONVERSATION, component: createView('chat').component }
    views.registerView('chat.plugin', registration)

    expect(() => views.registerView('chat.plugin', registration)).toThrow(/already draws/)
  })

  it('accepts another plugin once the one that drew the capability is disposed', () => {
    const views = createViewRegistry({ styles })
    const first = views.registerView('first.plugin', {
      capability: CONVERSATION,
      component: createView('first').component,
    })
    const element = document.createElement('section')
    views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' }, NO_ACCESS)

    first.dispose()
    expect(element.childElementCount).toBe(0)
    views.registerView('second.plugin', {
      capability: CONVERSATION,
      component: createView('second').component,
    })

    expect(element.textContent).toBe('second: Decks/Physics.md')
  })

  it('adds the stylesheet link of the declaring plugin before the view', () => {
    styles.clear()
    styles.register('chat.plugin', '/plugin/chat.plugin/dist/styles.css')
    const views = createViewRegistry({ styles })
    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const element = document.createElement('section')

    views.mountView(CONVERSATION, element, {}, NO_ACCESS)

    const link = element.querySelector('link')
    expect(link?.rel).toBe('stylesheet')
    expect(link?.getAttribute('href')).toBe('/plugin/chat.plugin/dist/styles.css')
    expect(link?.dataset['pluginId']).toBe('chat.plugin')
    styles.clear()
  })

  it('adopts a constructed stylesheet into the root and releases it on dispose', async () => {
    class FakeSheet {
      cssText = ''
      replaceSync(text: string) {
        this.cssText = text
      }
    }
    vi.stubGlobal('CSSStyleSheet', FakeSheet)
    Object.defineProperty(document, 'adoptedStyleSheets', {
      value: [],
      writable: true,
      configurable: true,
    })
    try {
      await styles.load(
        'chat.plugin',
        '/plugin/chat.plugin/dist/styles.css',
        async () => ({ ok: true, text: async () => '.chat {}' }) as unknown as Response,
      )
      const views = createViewRegistry({ styles })
      views.registerView('chat.plugin', {
        capability: CONVERSATION,
        component: createView('chat').component,
      })
      const element = document.createElement('section')
      document.body.appendChild(element)

      const mounted = views.mountView(CONVERSATION, element, { focus: 'Note.md' }, NO_ACCESS)
      expect(element.querySelector('link')).toBeNull()
      expect(element.querySelector('.plugin-view')).not.toBeNull()
      expect(document.adoptedStyleSheets.length).toBe(1)

      mounted.dispose()
      expect(document.adoptedStyleSheets.length).toBe(0)
      element.remove()
    } finally {
      styles.clear()
      vi.unstubAllGlobals()
      Reflect.deleteProperty(document, 'adoptedStyleSheets')
    }
  })

  it('draws nothing for a capability no declaration names', () => {
    const views = createViewRegistry({ styles })
    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const element = document.createElement('section')

    views.mountView('graph', element, {}, NO_ACCESS)

    expect(element.childElementCount).toBe(0)
  })

  it('starts declaring plugins only when the capability is not registered yet', () => {
    const bareManifest: Manifest = { ...createMockManifest('bare'), contributes: undefined }
    const emptyManifest = createMockManifest('empty')
    const matchingManifest = createMockManifest('chat.plugin', {
      viewCapabilities: [CONVERSATION],
    })
    const manifests = [bareManifest, emptyManifest, matchingManifest]
    const matchedIds: string[] = []
    const startPluginsOf = vi.fn(async (match: (m: Manifest) => boolean) => {
      for (const m of manifests) {
        if (match(m)) matchedIds.push(m.id)
      }
    })
    const views = createViewRegistry({ styles, startPluginsOf })
    const element = document.createElement('section')

    views.mountView(CONVERSATION, element, { focus: 'Note.md' }, NO_ACCESS)
    expect(startPluginsOf).toHaveBeenCalledOnce()
    expect(matchedIds).toStrictEqual(['chat.plugin'])

    startPluginsOf.mockClear()
    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const secondElement = document.createElement('section')
    views.mountView(CONVERSATION, secondElement, { focus: 'Second.md' }, NO_ACCESS)
    expect(startPluginsOf).not.toHaveBeenCalled()
  })
})
