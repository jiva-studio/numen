import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { CONVERSATION } from '@numen/plugin-sdk'
import { clearPluginStyles, registerPluginStyle } from './styles'
import { createViewRegistry } from './views'

/** A view that writes the context it was given, and counts the times the keyboard was put in it. */
function createView(name: string) {
  const focused = { count: 0 }
  const component = defineComponent({
    props: { context: { type: Object, required: true } },
    setup(props, { expose }) {
      expose({ focus: () => focused.count++ })
      return () => h('p', `${name}: ${(props.context as { focus: string }).focus}`)
    },
  })
  return { component, focused }
}

describe('createViewRegistry', () => {
  it('draws the declared view into the element with the context passed', () => {
    const views = createViewRegistry()
    const { component, focused } = createView('chat')
    views.registerView('chat.plugin', { capability: CONVERSATION, component })
    const element = document.createElement('section')

    const mounted = views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' })
    mounted.focus()

    expect(element.textContent).toBe('chat: Decks/Physics.md')
    expect(focused.count).toBe(1)
    mounted.dispose()
    expect(element.childElementCount).toBe(0)
  })

  it('leaves the element empty while no plugin declares the capability, and draws once one does', () => {
    const views = createViewRegistry()
    const element = document.createElement('section')

    const mounted = views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' })
    mounted.focus()
    expect(element.childElementCount).toBe(0)

    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    expect(element.textContent).toBe('chat: Decks/Physics.md')
  })

  it('takes the view away with the plugin that declared it', () => {
    const views = createViewRegistry()
    const declared = views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const element = document.createElement('section')
    views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' })

    declared.dispose()

    expect(element.childElementCount).toBe(0)
  })

  it('draws another declaration of the capability where the drawn one goes', () => {
    const views = createViewRegistry()
    const first = views.registerView('first.plugin', {
      capability: CONVERSATION,
      component: createView('first').component,
    })
    views.registerView('second.plugin', {
      capability: CONVERSATION,
      component: createView('second').component,
    })
    const element = document.createElement('section')
    views.mountView(CONVERSATION, element, { focus: 'Decks/Physics.md' })

    first.dispose()

    expect(element.textContent).toBe('second: Decks/Physics.md')
  })

  it('adds the stylesheet link of the declaring plugin before the view', () => {
    clearPluginStyles()
    registerPluginStyle('chat.plugin', '/api/plugins/chat.plugin/dist/styles.css')
    const views = createViewRegistry()
    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const element = document.createElement('section')

    views.mountView(CONVERSATION, element, {})

    const link = element.querySelector('link')
    expect(link?.getAttribute('href')).toBe('/api/plugins/chat.plugin/dist/styles.css')
    expect(link?.dataset['pluginId']).toBe('chat.plugin')
    clearPluginStyles()
  })

  it('draws nothing for a capability no declaration names', () => {
    const views = createViewRegistry()
    views.registerView('chat.plugin', {
      capability: CONVERSATION,
      component: createView('chat').component,
    })
    const element = document.createElement('section')

    views.mountView('graph', element, {})

    expect(element.childElementCount).toBe(0)
  })
})
