import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, type Component } from 'vue'
import { CONVERSATION, NOTE_EDITOR, VAULT_SUMMARY, type ViewCapability } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { STUB_CONNECTIONS } from '../testing/stubComponent'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

const READ_WRITE = ['vault:read', 'vault:write']

const createHost = () => {
  const host = createTestPluginHost({
    target: 'desktop.editor',
    registerKind: () => vi.fn(),
    registerExtensionReader: () => vi.fn(),
    openTab: () => Promise.resolve('tab'),
  } as unknown as TestHostDeps)
  const createPlugin = (
    id: string,
    options: { permissions?: string[]; viewCapabilities?: string[] } = {},
  ) =>
    host.lifecycle.createContext(
      id,
      createMockManifest(id, {
        permissions: options.permissions ?? READ_WRITE,
        viewCapabilities: options.viewCapabilities ?? [],
      }),
    )
  return { host, createPlugin }
}

const createView = (owner: string, received: string[]): Component =>
  defineComponent({
    props: { context: { type: Object, required: true } },
    setup(props) {
      received.push(owner)
      return () => h('p', `${owner}: ${(props.context as { vault: string }).vault}`)
    },
  })

const mountConversation = (
  plugin: ReturnType<ReturnType<typeof createHost>['createPlugin']>,
  element: HTMLElement,
) =>
  plugin.host.ui.mountView(CONVERSATION, element, { connections: STUB_CONNECTIONS, vault: 'roots' })

describe('a plugin that registers a view for other plugins', () => {
  it('is refused a capability its manifest does not list', () => {
    const { createPlugin } = createHost()
    const hijacker = createPlugin('org.example.hijacker')

    expect(() =>
      hijacker.plugin.register.view({ capability: CONVERSATION, component: createView('x', []) }),
    ).toThrowError(/contributes.viewCapabilities does not list/)
  })

  it('is refused a capability another plugin lists', () => {
    const { createPlugin } = createHost()
    const hijacker = createPlugin('org.example.hijacker', { viewCapabilities: [VAULT_SUMMARY] })

    expect(() =>
      hijacker.plugin.register.view({ capability: CONVERSATION, component: createView('x', []) }),
    ).toThrowError(/does not list/)
  })

  it('is refused without any manifest', () => {
    const { host } = createHost()

    expect(() =>
      host.lifecycle
        .createContext('org.example.bare')
        .plugin.register.view({ capability: CONVERSATION, component: createView('x', []) }),
    ).toThrowError(/does not list/)
  })

  it('never receives the context of a plugin that mounts the capability', () => {
    const { createPlugin } = createHost()
    const received: string[] = []
    const hijacker = createPlugin('org.example.hijacker')
    const element = document.createElement('div')
    mountConversation(createPlugin('md.numen.mounter'), element)

    expect(() =>
      hijacker.plugin.register.view({
        capability: CONVERSATION,
        component: createView('hijacker', received),
      }),
    ).toThrowError()

    expect(received).toStrictEqual([])
    expect(element.textContent).toBe('')
  })

  it('is refused a second registration of a capability, and the first view stays', () => {
    const { createPlugin } = createHost()
    const received: string[] = []
    const chat = createPlugin('md.numen.chat', { viewCapabilities: [CONVERSATION] })
    const second = createPlugin('org.example.second', { viewCapabilities: [CONVERSATION] })
    chat.plugin.register.view({ capability: CONVERSATION, component: createView('chat', received) })

    expect(() =>
      second.plugin.register.view({
        capability: CONVERSATION,
        component: createView('second', received),
      }),
    ).toThrowError(/"md.numen.chat" already draws/)
    const element = document.createElement('div')
    mountConversation(createPlugin('md.numen.mounter'), element)

    expect(received).toStrictEqual(['chat'])
    expect(element.textContent).toBe('chat: roots')
  })

  it('may register the capability once the plugin that drew it is disposed', () => {
    const { host, createPlugin } = createHost()
    createPlugin('md.numen.chat', {
      viewCapabilities: [CONVERSATION],
    }).plugin.identity.register.view({
      capability: CONVERSATION,
      component: createView('chat', []),
    })
    host.lifecycle.disposePlugin('md.numen.chat')
    const element = document.createElement('div')
    mountConversation(createPlugin('md.numen.mounter'), element)

    createPlugin('org.example.next', {
      viewCapabilities: [CONVERSATION],
    }).plugin.identity.register.view({
      capability: CONVERSATION,
      component: createView('next', []),
    })

    expect(element.textContent).toBe('next: roots')
  })

  it.each<[ViewCapability, string[], string]>([
    [NOTE_EDITOR, ['vault:read'], 'vault:write'],
    [NOTE_EDITOR, [], 'vault:read'],
    [VAULT_SUMMARY, [], 'vault:read'],
  ])(
    'is refused %s with the permissions %j, which lack %s the context hands over',
    (capability, permissions, missing) => {
      const { createPlugin } = createHost()
      const received: string[] = []
      const registrant = createPlugin('org.example.registrant', {
        permissions,
        viewCapabilities: [capability],
      })
      const element = document.createElement('div')
      createPlugin('md.numen.mounter').host.ui.mountView(capability, element, {
        connections: STUB_CONNECTIONS,
        path: 'Notes/Secret.md',
        vault: 'roots',
      })

      expect(() =>
        registrant.plugin.register.view({
          capability,
          component: createView('registrant', received),
        }),
      ).toThrowError(`lacks required permission "${missing}" for register.view`)

      expect(received).toStrictEqual([])
    },
  )

  it('registers a view of each kind for a plugin that holds the permissions of the kind', () => {
    const { createPlugin } = createHost()
    const cases = [
      [CONVERSATION, []],
      [NOTE_EDITOR, READ_WRITE],
      [VAULT_SUMMARY, ['vault:read']],
    ] as const

    for (const [index, [capability, permissions]] of cases.entries()) {
      const registrant = createPlugin(`org.example.registrant${index}`, {
        permissions: [...permissions],
        viewCapabilities: [capability],
      })
      expect(() =>
        registrant.plugin.register.view({ capability, component: createView('registrant', []) }),
      ).not.toThrow()
    }
  })

  it('is checked on one read of the registration, whatever a getter answers later', () => {
    const { createPlugin } = createHost()
    const received: string[] = []
    const hijacker = createPlugin('org.example.hijacker', { viewCapabilities: [VAULT_SUMMARY] })
    const answers = [VAULT_SUMMARY, CONVERSATION, CONVERSATION, CONVERSATION]
    const registration = {
      get capability() {
        return answers.shift() ?? CONVERSATION
      },
      component: createView('hijacker', received),
    }
    hijacker.plugin.register.view(registration as never)
    const element = document.createElement('div')

    mountConversation(createPlugin('md.numen.mounter'), element)

    expect(received).toStrictEqual([])
    expect(element.textContent).toBe('')
  })

  it('is checked on one read of a proxy that answers each read differently', () => {
    const { createPlugin } = createHost()
    const received: string[] = []
    const hijacker = createPlugin('org.example.hijacker', { viewCapabilities: [VAULT_SUMMARY] })
    let reads = 0
    const registration = new Proxy(
      { capability: VAULT_SUMMARY, component: createView('hijacker', received) },
      {
        get: (target, key) =>
          key === 'capability'
            ? reads++ === 0
              ? VAULT_SUMMARY
              : CONVERSATION
            : Reflect.get(target, key),
      },
    )
    hijacker.plugin.register.view(registration as never)
    const element = document.createElement('div')

    mountConversation(createPlugin('md.numen.mounter'), element)

    expect(received).toStrictEqual([])
    expect(element.textContent).toBe('')
  })

  it('leaves a waiting mount empty when its registration is refused, and draws the next valid one', () => {
    const { createPlugin } = createHost()
    const received: string[] = []
    const element = document.createElement('div')
    mountConversation(createPlugin('md.numen.mounter'), element)
    const hijacker = createPlugin('org.example.hijacker', { permissions: ['vault:read'] })

    expect(() =>
      hijacker.plugin.register.view({
        capability: CONVERSATION,
        component: createView('hijacker', received),
      }),
    ).toThrowError()
    createPlugin('md.numen.chat', {
      viewCapabilities: [CONVERSATION],
    }).plugin.identity.register.view({
      capability: CONVERSATION,
      component: createView('chat', received),
    })

    expect(received).toStrictEqual(['chat'])
    expect(element.textContent).toBe('chat: roots')
  })
})
