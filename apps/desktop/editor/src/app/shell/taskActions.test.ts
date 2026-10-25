import { describe, expect, it } from 'vitest'
import { isConfigurableError, resolveTaskPlugin, resolveTaskAction } from './taskActions'
import type { Task } from '@/shared/notices/task'

const PLUGINS = [
  { id: 'md.numen.agents.acp', name: 'ACP Agents' },
  { id: 'md.numen.agents.claude', name: 'Claude CLI' },
  { id: 'md.numen.engines.transcription', name: 'Audio/Video Transcription' },
  { id: 'md.numen.tools.proofreading', name: 'AI Proofreading' },
]

describe('isConfigurableError', () => {
  it('rejects file missing and filesystem errors', () => {
    expect(isConfigurableError('stat assets/sample.flac: no such file or directory')).toBe(false)
    expect(isConfigurableError('read Books/manual.pdf: file not found')).toBe(false)
    expect(isConfigurableError('stat /tmp/file: does not exist')).toBe(false)
    expect(isConfigurableError('open notes/today.md: permission denied')).toBe(false)
  })

  it('accepts authentication and configuration errors', () => {
    expect(isConfigurableError('Failed to authenticate: OAuth session expired')).toBe(true)
    expect(isConfigurableError('API key not configured')).toBe(true)
    expect(isConfigurableError('unauthorized: invalid credentials')).toBe(true)
    expect(isConfigurableError('quota exceeded for provider')).toBe(true)
  })
})

describe('resolveTaskPlugin', () => {
  it('does not falsely match ACP when error mentions generic agent word', () => {
    const task: Task = {
      id: 'task-1',
      label: 'Proofreading a recognition',
      about: 'Books/manual.pdf',
      completedCount: 0,
      total: 0,
      counting: 'items',
      error:
        'routed agent did not answer: the agent stopped: Failed to authenticate: OAuth session expired',
      isUserStarted: true,
    }

    const matched = resolveTaskPlugin(task, PLUGINS)
    // Should match proofreading rather than blindly defaulting to ACP
    expect(matched).toBe('md.numen.tools.proofreading')
  })

  it('matches specific agent when named in error', () => {
    const task: Task = {
      id: 'task-2',
      label: 'Agent run',
      about: 'chat',
      completedCount: 0,
      total: 0,
      counting: 'items',
      error: 'claude process failed to authenticate',
      isUserStarted: true,
    }

    expect(resolveTaskPlugin(task, PLUGINS)).toBe('md.numen.agents.claude')
  })
})

describe('resolveTaskAction', () => {
  it('returns undefined for file missing errors', () => {
    const task: Task = {
      id: 'task-missing',
      label: 'Transcribing a recording',
      about: 'assets/sample.flac',
      completedCount: 0,
      total: 0,
      counting: 'items',
      error: 'stat assets/sample.flac: no such file or directory',
      isUserStarted: false,
    }

    expect(resolveTaskAction(task, PLUGINS)).toBeUndefined()
  })

  it('returns configure action for authentication errors', () => {
    const task: Task = {
      id: 'task-auth',
      label: 'Proofreading a recognition',
      about: 'Books/manual.pdf',
      completedCount: 0,
      total: 0,
      counting: 'items',
      error: 'OAuth session expired and could not be refreshed',
      isUserStarted: false,
    }

    expect(resolveTaskAction(task, PLUGINS)).toEqual({
      label: 'Configure in settings',
      pluginId: 'md.numen.tools.proofreading',
    })
  })
})
