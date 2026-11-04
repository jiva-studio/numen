import { describe, expect, it } from 'vitest'
import {
  AGENT_USE_PATH,
  DEFAULT_AGENT_PATH,
  DEFAULT_MODEL_PATH,
  getProgramHooksPath,
  getProgramModelPath,
  getProgramStepsPath,
  readAgentChoice,
} from './agentSettings'

describe('agent settings paths', () => {
  it('names the setting paths for a program', () => {
    expect(AGENT_USE_PATH).toStrictEqual(['agent', 'use'])
    expect(DEFAULT_AGENT_PATH).toStrictEqual(['ai', 'default', 'agent'])
    expect(DEFAULT_MODEL_PATH).toStrictEqual(['ai', 'default', 'model'])
    expect(getProgramModelPath('claude')).toStrictEqual(['agent', 'claude', 'model'])
    expect(getProgramStepsPath('claude')).toStrictEqual(['agent', 'claude', 'max_steps'])
    expect(getProgramHooksPath('claude')).toStrictEqual([
      'agent',
      'claude',
      'reads_hooks_and_skills',
    ])
  })
})

describe('readAgentChoice', () => {
  it('returns empty strings when the raw setting is not an object', () => {
    expect(readAgentChoice(null)).toStrictEqual({ agent: '', model: '' })
    expect(readAgentChoice(undefined)).toStrictEqual({ agent: '', model: '' })
    expect(readAgentChoice('claude')).toStrictEqual({ agent: '', model: '' })
  })

  it('reads string agent and model fields and defaults non-strings to empty', () => {
    expect(readAgentChoice({ agent: 'claude', model: 'sonnet' })).toStrictEqual({
      agent: 'claude',
      model: 'sonnet',
    })
    expect(readAgentChoice({ agent: 42, model: true })).toStrictEqual({
      agent: '',
      model: '',
    })
  })
})
