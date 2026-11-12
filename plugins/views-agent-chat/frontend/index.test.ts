import { describe, expect, it } from 'vitest'
import { CONVERSATION, type TabKindRegistration, type VaultRelations } from '@numen/plugin-sdk'
import { createMockPluginContext, type MockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, AgentTab, AGENT, ConversationView } from './index'
import { createAgentConnections, createHost, say } from './fixtures'
import type { AgentTabState } from './model/useAgentConversation'

const PLUGIN_ID = 'md.numen.views.agent.chat'

const createWindow = () => {
  let getKinds = (): readonly TabKindRegistration[] => []
  const tabOpens: (readonly [string, string])[] = []
  const tabs: { id: string; kind: string; state: object }[] = []
  const navigateTab = async (kind: string, path: string): Promise<string> => {
    const registration = getKinds().find((one) => one.kind === kind)!
    const last = tabs.filter((one) => one.kind === kind).at(-1)
    if (last) {
      await registration.navigate?.(last.state, path, '')
      return last.id
    }
    tabOpens.push([kind, path])
    const id = `${kind}-${tabs.length}`
    tabs.push({ id, kind, state: registration.open!(path, id) })
    return id
  }
  const bind = (ctx: { getRegisteredTabKinds(): TabKindRegistration[] }): void => {
    getKinds = () => ctx.getRegisteredTabKinds()
  }
  return { tabOpens, tabs, navigateTab, bind }
}

/** A mock context of the editor, with the connections of the plugin and the links of the vault. */
const createContext = (
  answer: Parameters<typeof createAgentConnections>[0] = { frames: [say('Twelve.')] },
  navigateTab?: (kind: string, path: string) => Promise<string>,
): MockPluginContext => {
  const { workspace, relations } = createHost()
  const ctx = createMockPluginContext({
    pluginId: PLUGIN_ID,
    workspace,
    relations: relations as unknown as VaultRelations,
    ...(navigateTab ? { navigateTab } : {}),
  })
  const { host } = createAgentConnections(answer)
  return { ...ctx, work: { ...ctx.work, connections: host } }
}

describe('view-agent-chat plugin', () => {
  it('registers the agent tab kind upon activation', () => {
    const ctx = createContext()
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(AGENT)
    expect(tabKinds[0]?.component).toBe(AgentTab)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('declares the conversation view other plugins mount', () => {
    const ctx = createContext()
    activate(ctx)

    expect(ctx.getRegisteredViews()).toStrictEqual([
      { capability: CONVERSATION, component: ConversationView },
    ])
  })

  it('declares only the conversation view in the flashcards window, which offers no tabs', () => {
    const ctx = createMockPluginContext({ pluginId: PLUGIN_ID, target: 'desktop.flashcards' })
    activate(ctx)

    expect(ctx.getRegisteredViews().map((one) => one.capability)).toStrictEqual([CONVERSATION])
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('removes the agent tab kind when the plugin is disabled', () => {
    const ctx = createContext()
    activate(ctx)

    ctx.dispose()

    expect(ctx.plugin.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredViews()).toHaveLength(0)
  })

  it('opens a tab whose conversation is its own', () => {
    const ctx = createContext()
    activate(ctx)
    const kind = ctx.getRegisteredTabKinds()[0]!

    const first = kind.open!('', 'agent:1') as AgentTabState
    const second = kind.open!('', 'agent:2') as AgentTabState
    kind.receiveText!(first, 'what is this?')

    expect(first.draft.userQuestion.value).toBe('what is this?')
    expect(second.draft.userQuestion.value).toBe('')
  })

  it('answers a question of a tab through the connections of the plugin', async () => {
    const ctx = createContext({ frames: [say('Twelve pear trees.')] })
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'agent:1') as AgentTabState

    state.draft.send('How many pear trees?')
    while (state.conversation.isAnswerPending.value) await new Promise((go) => setTimeout(go, 0))

    expect(state.addresses.turns.value.map((turn) => turn.text)).toContain('Twelve pear trees.')
  })

  it('opens the settings page of the plugin from a tab', () => {
    const ctx = createContext()
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'agent:1') as AgentTabState

    state.openSettings()

    expect(ctx.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'openSettings', id: PLUGIN_ID },
    ])
  })

  it('opens one conversation however many times the command runs', async () => {
    const strip = createWindow()
    const ctx = createContext(undefined, strip.navigateTab)
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

  it('writes the path of the note into the composer of the conversation last in use', async () => {
    const strip = createWindow()
    const ctx = createContext(undefined, strip.navigateTab)
    strip.bind(ctx)
    activate(ctx)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
      'chat.open',
      'note.ask',
    ])
    await ctx.triggerCommand('chat.open')
    await ctx.triggerCommand('note.ask', {
      path: 'notes/Orchard.md',
      title: 'Orchard',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(strip.tabOpens).toStrictEqual([[AGENT, '']])
    const state = strip.tabs[0]!.state as AgentTabState
    expect(state.draft.userQuestion.value).toBe('notes/Orchard.md — ')
  })

  it('opens a conversation for a question about a note where none is open', async () => {
    const strip = createWindow()
    const ctx = createContext(undefined, strip.navigateTab)
    strip.bind(ctx)
    activate(ctx)

    await ctx.triggerCommand('note.ask', {
      path: 'notes/Orchard.md',
      title: 'Orchard',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(strip.tabs).toHaveLength(1)
    const state = strip.tabs[0]!.state as AgentTabState
    expect(state.draft.userQuestion.value).toBe('notes/Orchard.md — ')
  })

  it('holds the conversation view, the tab kind and its two commands, and nothing else', () => {
    const ctx = createContext()
    activate(ctx)

    expect(ctx.plugin.subscriptions).toHaveLength(4)
  })

  it('refuses to activate in a window that offers no connections', () => {
    const { workspace, relations } = createHost()
    expect(() =>
      activate(
        createMockPluginContext({
          pluginId: PLUGIN_ID,
          workspace,
          relations: relations as unknown as VaultRelations,
        }),
      ),
    ).toThrow('an agent chat requires a window, its connections and the links of the vault')
  })

  it('activates through the default plugin lifecycle', async () => {
    const ctx = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(AGENT)
  })
})
