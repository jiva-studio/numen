import { describe, expect, it } from 'vitest'
import { isConfigurableError, resolveTaskAction } from './taskActions'
import type { Task } from '@/shared/notices/task'

const failed: Task = {
  id: 'task',
  label: 'Working',
  about: 'notes/one.md',
  completedCount: 0,
  total: 0,
  counting: 'items',
  error: 'OAuth session expired and could not be refreshed',
  isUserStarted: false,
  pluginId: 'md.example.engine',
  pluginName: 'Example Engine',
}

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

describe('resolveTaskAction', () => {
  it('returns undefined for file missing errors', () => {
    const task = { ...failed, error: 'stat assets/sample.flac: no such file or directory' }

    expect(resolveTaskAction(task)).toBeUndefined()
  })

  it('returns the configure action for the plugin that reported the task', () => {
    expect(resolveTaskAction(failed)).toEqual({
      label: 'Configure in settings',
      pluginId: 'md.example.engine',
    })
  })

  it('returns undefined for a core task', () => {
    expect(resolveTaskAction({ ...failed, pluginId: '', pluginName: '' })).toBeUndefined()
  })
})
