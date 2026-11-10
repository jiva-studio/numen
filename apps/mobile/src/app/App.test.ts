/**
 * What the application opens, with the native half and the core's side of the wire stood in for.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { NoteSheet } from '@/pages/note-sheet'
import App from './App.vue'

const encoder = new TextEncoder()

/** A string field of a message. */
const text = (field: number, value: string) => [
  (field << 3) | 2,
  value.length,
  ...encoder.encode(value),
]

/** A message field, holding `parts`. */
const message = (field: number, ...parts: number[][]) => {
  const body = parts.flat()
  return [(field << 3) | 2, body.length, ...body]
}

vi.mock('@capacitor/core', () => ({
  registerPlugin: () => ({ start: async () => ({ port: 45123, dir: '/data' }) }),
}))

/** Answers each service call of the core by its method name. */
function answerWith(answers: Record<string, number[]>) {
  vi.stubGlobal('fetch', async (input: RequestInfo | URL) => {
    const method = String(input instanceof Request ? input.url : input).split('/').pop() ?? ''
    return new Response(new Uint8Array(answers[method] ?? []), {
      headers: { 'content-type': 'application/proto' },
    })
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('a core that answers', () => {
  it('opens the opening note, and opens it again from its row once closed', async () => {
    answerWith({
      GetOpeningNote: message(1, text(1, 'Physics.md')),
      ReadNote: text(1, '# Physics\n'),
    })
    const app = mount(App, { attachTo: document.body })
    await flushPromises()

    expect(app.getComponent(NoteSheet).props('path')).toBe('Physics.md')
    expect(app.get('textarea').element.value).toBe('# Physics\n')

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
