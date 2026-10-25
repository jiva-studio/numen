import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { formatErrorMessage } from '@numen/wire'

const { getPlugins } = vi.hoisted(() => ({ getPlugins: vi.fn() }))
vi.mock('@/shared/clients', () => ({ pluginService: { getPlugins } }))

const { default: App } = await import('./App.vue')

describe('App', () => {
  it('says why it draws nothing when the plugins cannot be listed', async () => {
    getPlugins.mockRejectedValueOnce(new Error('the window is gone'))

    const window = mount(App)
    await flushPromises()

    expect(window.text()).toContain('No plugin of this window draws anything.')
    expect(window.text()).toContain(formatErrorMessage(new Error('the window is gone')))
    window.unmount()
  })
})

describe('useShell', () => {
  it('draws the view the loaded plugin registers, with its stylesheet', async () => {
    const { useShell } = await import('./useShell')
    const Review = defineComponent({ render: () => h('p', 'the review') })
    const manifest = {
      id: 'md.numen.example',
      name: 'Example',
      version: '1.0.0',
      frontend: { entrypoint: 'dist/index.js', styles: 'dist/styles.css' },
    }
    const fetch = vi.fn(async () => new Response(JSON.stringify(manifest)))
    const dynamicImport = vi.fn(async () => ({
      activate: (context: { registerTabKind: (one: unknown) => void }) =>
        context.registerTabKind({
          kind: 'review',
          component: Review,
          placement: { pane: 'main', order: 0 },
        }),
    }))
    let shell: ReturnType<typeof useShell> | undefined
    const window = mount(
      defineComponent({
        setup: () => {
          shell = useShell({
            listPlugins: async () => ['md.numen.example'],
            fetch: fetch as unknown as typeof globalThis.fetch,
            dynamicImport,
          })
          return () => null
        },
      }),
    )
    await flushPromises()

    expect(shell?.front.value?.registration.kind).toBe('review')
    expect(shell?.style.value).toBe('/api/plugins/md.numen.example/dist/styles.css')
    expect(dynamicImport).toHaveBeenCalledWith(
      expect.stringMatching(/^\/api\/plugins\/md\.numen\.example\/dist\/index\.js\?activation=/),
    )
    window.unmount()
  })
})
