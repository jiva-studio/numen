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

  it('declares a new conversation over the window, and a question about a note', async () => {
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

    expect(
      ctx.getRegisteredCommands().map((one) => [one.id, one.scope, one.shortcut]),
    ).toStrictEqual([
      ['agent', undefined, 'mod+shift+a'],
      ['ask', 'note', undefined],
    ])
    await ctx.triggerCommand('agent')
    await ctx.triggerCommand('ask', { path: 'One.md', title: 'One', type: 'note' })
    await ctx.getRegisteredContextMenus()[0]!.handler({
      path: 'Two.md',
      name: 'Two',
      isFolder: false,
      isNote: true,
    })

    expect(openCalls).toStrictEqual([[AGENT, undefined]])
    expect(recorded.questions).toStrictEqual(['One.md — ', 'Two.md — '])
  })

  it('refuses to activate in a window that offers no workspace or agent', () => {
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.views.agent.chat' })),
    ).toThrow('an agent chat needs the window it is drawn in and the agent it asks')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(AGENT)
  })
})
