/**
 * Which agents and models an agent tab offers, and which of them it starts on.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '@numen/ui'
import type { Model } from '@/entities/settings'
import { useAgentConversation } from './useAgentConversation'

/** A conversation with nothing said in it. */
const createConversation = (): Conversation => ({
  turns: ref([]),
  isWorking: ref(false),
  ask: async () => {},
  getSourceLocation: () => null,
  stop: () => {},
  finish: () => {},
})

/** A setting the window lists, found at the place given. */
const model = (namedAt: readonly string[], name: string, presence: Model['presence']): Model => ({
  namedAt,
  name,
  title: name,
  shelf: '',
  isDefault: false,
  writes: [],
  presence,
})

/** An agent tab over the models listed and the settings kept. */
const open = (models?: readonly Model[], settings: Record<string, string> = {}) =>
  useAgentConversation(createConversation(), {
    openFileAt: () => {},
    openFileBeside: () => {},
    resolve: async () => new Map(),
    unreachable: () => '',
    getSetting: (path) => settings[path.join('/')],
    ...(models ? { getModels: () => models } : {}),
  })

/** The agents the tab offers, each once, in the order offered. */
const offered = (state: ReturnType<typeof open>) => [
  ...new Set(state.modelOptions.value.map((one) => one.agentId)),
]

describe('the model an agent tab starts on', () => {
  it('is the one kept for the agent that was kept', () => {
    const state = open(undefined, {
      'agent/use': 'claude',
      'agent/claude/model': 'opus',
    })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual(['claude', 'opus'])
  })

  it('is the one kept for the first agent where no agent was kept', () => {
    const state = open(undefined, { 'agent/antigravity/model': 'gemini-3.1-pro-high' })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'antigravity',
      'gemini-3.1-pro-high',
    ])
  })

  it('is the first model offered where the one kept is not offered by that agent', () => {
    const state = open(undefined, {
      'agent/use': 'claude',
      'agent/claude/model': 'no-such-model',
    })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'antigravity',
      'gemini-3.7-flash-high',
    ])
  })

  it('is the first model offered where the model kept belongs to another agent', () => {
    const state = open(undefined, { 'agent/use': 'codex', 'agent/codex/model': 'sonnet' })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'antigravity',
      'gemini-3.7-flash-high',
    ])
  })
})

describe('the agents an agent tab offers', () => {
  it('are those listed under agent/use that are present', () => {
    const state = open([
      model(['agent', 'use'], 'claude', 'present'),
      model(['agent', 'use'], 'codex', 'not fetched'),
    ])

    expect(offered(state)).toEqual(['claude'])
  })

  it('are all of them where nothing is listed under agent/use', () => {
    const state = open([model(['editor', 'font'], 'claude', 'present')])

    expect(offered(state)).toEqual(['antigravity', 'claude', 'codex'])
  })

  it('leave out a setting listed under another name', () => {
    const state = open([
      model(['agent', 'use'], 'claude', 'present'),
      model(['agent', 'claude', 'model'], 'codex', 'present'),
    ])

    expect(offered(state)).toEqual(['claude'])
  })

  it('leave out a setting listed deeper than agent/use', () => {
    const state = open([
      model(['agent', 'use'], 'claude', 'present'),
      model(['agent', 'use', 'more'], 'codex', 'present'),
    ])

    expect(offered(state)).toEqual(['claude'])
  })

  it('leave out a setting listed under another group', () => {
    const state = open([
      model(['agent', 'use'], 'claude', 'present'),
      model(['other', 'use'], 'antigravity', 'present'),
    ])

    expect(offered(state)).toEqual(['claude'])
  })

  it('leave out a setting listed at agent under another key', () => {
    const state = open([
      model(['agent', 'use'], 'claude', 'present'),
      model(['agent', 'model'], 'codex', 'present'),
    ])

    expect(offered(state)).toEqual(['claude'])
  })
})
