import { describe, expect, it } from 'vitest'
import { CONVERSATION, type TabKindRegistration } from '@numen/plugin-sdk'
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

const createWindow = () => {
  let getKinds = (): readonly TabKindRegistration[] => []
  const tabOpens: (readonly [string, string])[] = []
  const tabs: { kind: string; state: object }[] = []
  const navigateTab = async (kind: string, path: string): Promise<string> => {
    const registration = getKinds().find((one) => one.kind === kind)!
    const last = tabs.filter((one) => one.kind === kind).at(-1)
    if (last) {
      await registration.navigate?.(last.state, path, '')
      return kind
    }
    tabOpens.push([kind, path])
    tabs.push({ kind, state: registration.open!(path, `${kind}-${tabs.length}`) })
    return kind
  }
  const bind = (ctx: { getRegisteredTabKinds(): TabKindRegistration[] }): void => {
    getKinds = () => ctx.getRegisteredTabKinds()
  }
  return { tabOpens, tabs, navigateTab, bind }
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

    expect(ctx.plugin.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredViews()).toHaveLength(0)
  })

  it('opens a tab whose conversation is its own', () => {
    const { ctx } = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!

    const first = kind.open!('', 'agent:1') as { draft: { userQuestion: { value: string } } }
    const second = kind.open!('', 'agent:2') as { draft: { userQuestion: { value: string } } }
    kind.receiveText!(first, 'what is this?')

    expect(first.draft.userQuestion.value).toBe('what is this?')
    expect(second.draft.userQuestion.value).toBe('')
  })

  it('opens one conversation however many times the command runs', async () => {
    const { host } = createHost()
    const strip = createWindow()
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.agent.chat',
      workspace: host.workspace,
      agent: host.agent,
      navigateTab: strip.navigateTab,
    })
    strip.bind(ctx)
    activate(ctx)

    await ctx.triggerCommand('chat.open')
    await ctx.triggerCommand('chat.open', {
      path: 'One.md',
      title: 'One.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(strip.tabOpens).toStrictEqual([[AGENT, '']])
    expect(strip.tabs).toHaveLength(1)
  })

  it('registers the handlers that open a conversation and ask about a note', async () => {
    const { host, recorded } = createHost()
    const strip = createWindow()
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.agent.chat',
      workspace: host.workspace,
      agent: host.agent,
      navigateTab: strip.navigateTab,
    })
    strip.bind(ctx)
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

    expect(strip.tabOpens).toStrictEqual([[AGENT, '']])
    expect(recorded.questions).toStrictEqual(['One.md — '])
  })

  it('holds the conversation view, the tab kind and its two commands, and nothing else', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.plugin.subscriptions).toHaveLength(4)
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
