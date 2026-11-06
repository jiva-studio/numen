import { describe, expect, it } from 'vitest'
import { CONVERSATION } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, AgentTab, AGENT, ConversationView } from './index'
import { createHost } from './fixtures'

const createContext = () => {
  const { host, recorded } = createHost()
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.views.agent.chat',
    workspace: host.workspace,
    agent: host.agent,
  })
  return { ctx, recorded }
}

describe('view-agent-chat plugin', () => {
  it('registers the agent tab kind upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(AGENT)
    expect(tabKinds[0]?.component).toBe(AgentTab)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('declares the conversation view other plugins mount', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredViews()).toStrictEqual([
      { capability: CONVERSATION, component: ConversationView },
    ])
  })

  it('declares only the conversation view in the flashcards window, which offers no tabs', () => {
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.agent.chat',
      target: 'desktop.flashcards',
    })
    activate(ctx)

    expect(ctx.getRegisteredViews().map((one) => one.capability)).toStrictEqual([CONVERSATION])
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('removes the agent tab kind when the plugin is disabled', () => {
    const { ctx } = createContext()
    activate(ctx)

    ctx.dispose()

    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredViews()).toHaveLength(0)
  })

  it('opens a tab whose conversation is its own', () => {
    const { ctx } = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!

    const first = kind.open!('', 'agent:1') as { userQuestion: { value: string } }
    const second = kind.open!('', 'agent:2') as { userQuestion: { value: string } }
    kind.receiveText!(first, 'what is this?')

    expect(first.userQuestion.value).toBe('what is this?')
    expect(second.userQuestion.value).toBe('')
  })

  it('registers the handlers that open a conversation and ask about a note', async () => {
    const openCalls: (readonly [string, string | undefined])[] = []
    const { host, recorded } = createHost()
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.agent.chat',
      workspace: host.workspace,
      agent: host.agent,
      openTab: (kind, path) => {
        openCalls.push([kind, path])
        return Promise.resolve('')
      },
    })
    activate(ctx)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
      'chat.open',
      'note.ask',
    ])
    await ctx.triggerCommand('chat.open')
    await ctx.triggerCommand('note.ask', {
      path: 'One.md',
      title: 'One',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(openCalls).toStrictEqual([[AGENT, undefined]])
    expect(recorded.questions).toStrictEqual(['One.md — '])
  })

  it('holds the conversation view, the tab kind and its two commands, and nothing else', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.subscriptions).toHaveLength(4)
  })

  it('refuses to activate in a window that offers no workspace or agent', () => {
    const { host } = createHost()
    expect(() =>
      activate(
        createMockPluginContext({
          pluginId: 'md.numen.views.agent.chat',
          workspace: host.workspace,
        }),
      ),
    ).toThrow('an agent chat requires a window and an agent')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(AGENT)
  })
})
