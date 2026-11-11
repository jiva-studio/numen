/**
 * What the application opens, with the native half and the core's side of the wire stood in for.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { NoteSheet } from '@/pages/note-sheet'
import App from './App.vue'

vi.mock('@capacitor/core', () => ({
  registerPlugin: () => ({ start: async () => ({ port: 45123, dir: '/data' }) }),
}))

function answerWith(answers: Record<string, object>) {
  vi.stubGlobal('fetch', async (input: RequestInfo | URL) => {
    const method =
      String(input instanceof Request ? input.url : input)
        .split('/')
        .pop() ?? ''
    return new Response(JSON.stringify(answers[method] ?? {}), {
      headers: { 'content-type': 'application/json' },
    })
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('a core that answers', () => {
  it('opens the opening note, and opens it again from its row once closed', async () => {
    answerWith({
      GetOpeningNote: { note: { path: 'Physics.md' } },
      ReadNote: { body: '# Physics\n' },
    })
    const app = mount(App, { attachTo: document.body })
    await flushPromises()

    expect(app.getComponent(NoteSheet).props('path')).toBe('Physics.md')
    expect(document.querySelector('textarea')?.value).toBe('# Physics\n')

    app.getComponent(NoteSheet).vm.$emit('close')
    await flushPromises()
    expect(app.findComponent(NoteSheet).exists()).toBe(false)

    await app.get('[data-testid="note"]').trigger('click')
    await flushPromises()
    expect(app.getComponent(NoteSheet).props('path')).toBe('Physics.md')

    app.unmount()
  })

  it('opens no note and shows no row for a vault without an opening note', async () => {
    answerWith({})
    const app = mount(App, { attachTo: document.body })
    await flushPromises()

    expect(app.findComponent(NoteSheet).exists()).toBe(false)
    expect(app.find('[data-testid="note"]').exists()).toBe(false)

    app.unmount()
  })
})
