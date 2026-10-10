import { describe, expect, it } from 'vitest'
import { calculateAggregate, formatEta, groupTasks } from './aggregate'
import type { StatusBarTask } from '../model/types'

describe('calculateAggregate', () => {
  it('returns idle state when no tasks are present', () => {
    const aggregate = calculateAggregate([])
    expect(aggregate.isIdle).toBe(true)
    expect(aggregate.activeCount).toBe(0)
    expect(aggregate.failedCount).toBe(0)
    expect(aggregate.overallProgress).toBeUndefined()
    expect(aggregate.etaSeconds).toBeUndefined()
  })

  it('calculates average progress across multiple determinate active tasks', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', group: 'Books', progress: 40, state: 'running' },
      { id: '2', label: 'Book 2', group: 'Books', progress: 80, state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.failedCount).toBe(0)
    expect(aggregate.overallProgress).toBe(60)
  })

  it('handles mixed determinate and indeterminate tasks', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', group: 'Books', progress: 50, state: 'running' },
      { id: '2', label: 'Scan', group: 'Vault', state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.overallProgress).toBe(50)
  })

  it('returns undefined progress when all tasks are indeterminate', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Scan 1', group: 'Vault', state: 'running' },
      { id: '2', label: 'Scan 2', group: 'Vault', state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.overallProgress).toBeUndefined()
  })

  it('counts failed tasks and treats them as non-idle', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', group: 'Books', state: 'failed', errorReason: 'Corrupted' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(0)
    expect(aggregate.failedCount).toBe(1)
  })

  it('finds maximum ETA among active tasks', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', group: 'Books', state: 'running', etaSeconds: 30 },
      { id: '2', label: 'Audio 1', group: 'Audio', state: 'running', etaSeconds: 120 },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.etaSeconds).toBe(120)
  })
})

describe('groupTasks', () => {
  it('groups tasks by their group name in order of first occurrence', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Task 1', group: 'Books', state: 'running' },
      { id: '2', label: 'Task 2', group: 'Audio', state: 'running' },
      { id: '3', label: 'Task 3', group: 'Books', state: 'running' },
    ]
    const groups = groupTasks(tasks)
    expect(groups).toHaveLength(2)
    expect(groups[0]?.name).toBe('Books')
    expect(groups[0]?.tasks).toHaveLength(2)
    expect(groups[1]?.name).toBe('Audio')
    expect(groups[1]?.tasks).toHaveLength(1)
  })
})

describe('formatEta', () => {
  it('returns empty string for missing or non-positive values', () => {
    expect(formatEta(undefined)).toBe('')
    expect(formatEta(0)).toBe('')
    expect(formatEta(-5)).toBe('')
  })

  it('formats seconds', () => {
    expect(formatEta(45)).toBe('~45s')
  })

  it('formats minutes', () => {
    expect(formatEta(90)).toBe('~2m')
    expect(formatEta(300)).toBe('~5m')
  })

  it('formats hours', () => {
    expect(formatEta(3600)).toBe('~1.0h')
    expect(formatEta(5400)).toBe('~1.5h')
  })
})
