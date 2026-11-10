import { describe, expect, it } from 'vitest'
import {
  AGENT_USE_PATH,
  DEFAULT_AGENT_PATH,
  DEFAULT_MODEL_PATH,
  getProgramModelPath,
  parseAgentChoice,
} from './agentSettings'

describe('agent settings paths', () => {
  it('names the setting paths for a program', () => {
    expect(AGENT_USE_PATH).toStrictEqual(['app', 'agent', 'use'])
    expect(DEFAULT_AGENT_PATH).toStrictEqual(['app', 'ai', 'default', 'agent'])
    expect(DEFAULT_MODEL_PATH).toStrictEqual(['app', 'ai', 'default', 'model'])
    expect(getProgramModelPath('claude')).toStrictEqual(['app', 'agent', 'claude', 'model'])
  })
})

describe('parseAgentChoice', () => {
  it('returns empty strings when the raw setting is not an object', () => {
    expect(parseAgentChoice(null)).toStrictEqual({ agent: '', model: '' })
    expect(parseAgentChoice(undefined)).toStrictEqual({ agent: '', model: '' })
    expect(parseAgentChoice('claude')).toStrictEqual({ agent: '', model: '' })
  })

  it('reads string agent and model fields and defaults non-strings to empty', () => {
    expect(parseAgentChoice({ agent: 'claude', model: 'sonnet' })).toStrictEqual({
      agent: 'claude',
      model: 'sonnet',
    })
    expect(parseAgentChoice({ agent: 42, model: true })).toStrictEqual({
      agent: '',
      model: '',
    })
  })
})
