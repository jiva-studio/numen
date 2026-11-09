import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref, type PropType } from 'vue'
import type { VaultNotes } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

const VIEW = 'md.numen.owner.note-viewer'

const settle = async () => {
  await new Promise((resolve) => setTimeout(resolve, 20))
  await nextTick()
}

const createHost = () => {
  const read = vi.fn(async (path: string) => ({ body: `Body of ${path}` }))
  const host = createPluginHost({
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve('tab'),
    getNotes: () => ({ read }) as unknown as VaultNotes,
  } as unknown as PluginHostDeps)
  const createPlugin = (id: string, permissions: string[]) =>
    host.createContext(id, createMockManifest(id, { name: id, permissions }))
  return { read, createPlugin }
}

type Plugin = ReturnType<ReturnType<typeof createHost>['createPlugin']>

// A view of the registering plugin that reads context.path through the notes service of
// that plugin, like the note editor view does.
const registerNoteViewer = (owner: Plugin, capability = VIEW) =>
  owner.registerView({
    capability,
    component: defineComponent({
      props: { context: { type: Object as PropType<{ path: string }>, required: true } },
      setup(props) {
        const body = ref('')
        owner.notes!.read(props.context.path).then(
          (one) => (body.value = one.body),
          (error: unknown) => (body.value = `Failed: ${String(error)}`),
        )
        return () => h('div', body.value)
      },
    }),
  })

describe('a view kind a plugin registers itself', () => {
  it('refuses to mount it to a plugin without the vault permission its plugin holds', async () => {
    const { read, createPlugin } = createHost()
    registerNoteViewer(createPlugin('md.numen.owner', ['vault:read']))
    const element = document.createElement('div')

    expect(() =>
      createPlugin('md.numen.caller', []).mountView(VIEW, element, { path: 'Notes/Example.md' }),
    ).toThrow(/vault:read/)
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount it to a plugin that lacks one of several vault permissions its plugin holds', () => {
    const { createPlugin } = createHost()
    registerNoteViewer(createPlugin('md.numen.owner', ['vault:read', 'vault:write']))

    expect(() =>
      createPlugin('md.numen.caller', ['vault:read']).mountView(
        VIEW,
        document.createElement('div'),
        { path: 'Notes/Example.md' },
      ),
    ).toThrow(/vault:write/)
  })

  it('mounts it for a plugin that holds the vault permission its plugin holds', async () => {
    const { createPlugin } = createHost()
    registerNoteViewer(createPlugin('md.numen.owner', ['vault:read']))
    const element = document.createElement('div')

    createPlugin('md.numen.caller', ['vault:read']).mountView(VIEW, element, {
      path: 'Notes/Example.md',
    })

    await vi.waitFor(() => expect(element.textContent).toBe('Body of Notes/Example.md'))
  })

  it('mounts it for any plugin when its plugin holds no vault permission', () => {
    const { createPlugin } = createHost()
    const owner = createPlugin('md.numen.owner', ['network'])
    owner.registerView({
      capability: VIEW,
      component: defineComponent({ setup: () => () => h('p', 'plain') }),
    })
    const element = document.createElement('div')

    expect(() => createPlugin('md.numen.caller', []).mountView(VIEW, element, {})).not.toThrow()
    expect(element.textContent).toBe('plain')
  })

  it('mounts it for the plugin that registered it', async () => {
    const { createPlugin } = createHost()
    const owner = createPlugin('md.numen.owner', ['vault:read'])
    registerNoteViewer(owner)
    const element = document.createElement('div')

    owner.mountView(VIEW, element, { path: 'Notes/Example.md' })

    await vi.waitFor(() => expect(element.textContent).toBe('Body of Notes/Example.md'))
  })
})

describe('a view capability named like a property of every object', () => {
  const names = ['constructor', '__proto__', 'toString', 'hasOwnProperty', 'valueOf']

  it.each(names)('waits for a registration when "%s" is mounted before one exists', async (name) => {
    const { createPlugin } = createHost()
    const caller = createPlugin('md.numen.caller', [])
    const element = document.createElement('div')

    expect(() => caller.mountView(name, element, {})).not.toThrow()
    createPlugin('md.numen.owner', []).registerView({
      capability: name,
      component: defineComponent({ setup: () => () => h('p', 'late') }),
    })
    await settle()

    expect(element.textContent).toBe('late')
  })

  it.each(names)('refuses a plugin without vault:read to mount the registered "%s"', async (name) => {
    const { read, createPlugin } = createHost()
    registerNoteViewer(createPlugin('md.numen.owner', ['vault:read']), name)
    const element = document.createElement('div')

    expect(() =>
      createPlugin('md.numen.caller', []).mountView(name, element, { path: 'Notes/Example.md' }),
    ).toThrow(/vault:read/)
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })
})
