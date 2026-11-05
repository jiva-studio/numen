import { describe, expect, it } from 'vitest'
import { calculateAggregate, formatEta, orderTasks } from './aggregate'
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
      { id: '1', label: 'Book 1', progress: 40, state: 'running' },
      { id: '2', label: 'Book 2', progress: 80, state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.failedCount).toBe(0)
    expect(aggregate.overallProgress).toBe(60)
  })

  it('handles mixed determinate and indeterminate tasks', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', progress: 50, state: 'running' },
      { id: '2', label: 'Scan', state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.overallProgress).toBe(50)
  })

  it('returns undefined progress when all tasks are indeterminate', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Scan 1', state: 'running' },
      { id: '2', label: 'Scan 2', state: 'running' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(2)
    expect(aggregate.overallProgress).toBeUndefined()
  })

  it('counts failed tasks and treats them as non-idle', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', state: 'failed', error: 'Corrupted' },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.activeCount).toBe(0)
    expect(aggregate.failedCount).toBe(1)
  })

  it('finds maximum ETA among active tasks', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Book 1', state: 'running', etaSeconds: 30 },
      { id: '2', label: 'Audio 1', state: 'running', etaSeconds: 120 },
    ]
    const aggregate = calculateAggregate(tasks)
    expect(aggregate.etaSeconds).toBe(120)
  })
})

describe('orderTasks', () => {
  it('returns an empty list when no tasks are present', () => {
    expect(orderTasks([])).toEqual([])
  })

  it('orders tasks newest first without mutating the input list', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Task 1', state: 'running' },
      { id: '2', label: 'Task 2', state: 'running' },
      { id: '3', label: 'Task 3', state: 'running' },
    ]
    const ordered = orderTasks(tasks)
    expect(ordered.map((one) => one.id)).toEqual(['3', '2', '1'])
    expect(tasks.map((one) => one.id)).toEqual(['1', '2', '3'])
  })
})

describe('calculateAggregate eta and cancelling', () => {
  const createTask = (overrides: Partial<StatusBarTask>): StatusBarTask => ({
    id: 'x',
    label: 'x',
    state: 'running',
    ...overrides,
  })

  it('counts a cancelling task as active, with its progress and eta', () => {
    const aggregate = calculateAggregate([
      createTask({ state: 'cancelling', progress: 30, etaSeconds: 20 }),
    ])
    expect(aggregate.activeCount).toBe(1)
    expect(aggregate.isIdle).toBe(false)
    expect(aggregate.overallProgress).toBe(30)
    expect(aggregate.etaSeconds).toBe(20)
  })

  it('takes the longest eta of the active tasks', () => {
    const aggregate = calculateAggregate([
      createTask({ etaSeconds: 100 }),
      createTask({ etaSeconds: 30 }),
      createTask({ etaSeconds: 60 }),
    ])
    expect(aggregate.etaSeconds).toBe(100)
  })

  it('takes the eta of a task that has one alongside tasks that have none', () => {
    const aggregate = calculateAggregate([createTask({}), createTask({ etaSeconds: 45 })])
    expect(aggregate.etaSeconds).toBe(45)
  })

  it('ignores an eta that is not positive', () => {
    expect(calculateAggregate([createTask({ etaSeconds: -5 })]).etaSeconds).toBeUndefined()
    expect(calculateAggregate([createTask({ etaSeconds: 0 })]).etaSeconds).toBeUndefined()
    expect(
      calculateAggregate([createTask({ etaSeconds: -5 }), createTask({ etaSeconds: 7 })])
        .etaSeconds,
    ).toBe(7)
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

  it('formats a full minute as a minute', () => {
    expect(formatEta(59)).toBe('~59s')
    expect(formatEta(60)).toBe('~1m')
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

describe('orderTasks across reports', () => {
  const report = (step: number): StatusBarTask[] => [
    { id: 'plugin-a', label: 'Reading', progress: step, state: 'running' },
    { id: 'vectors', label: 'Indexing', about: `file ${step}`, progress: step, state: 'running' },
    { id: 'plugin-b', label: 'Scanning', about: `page ${step}`, state: 'running' },
    { id: 'walk', label: 'Walking', state: 'running' },
  ]
  const ids = (list: readonly StatusBarTask[]): string[] => list.map((one) => one.id)

  it('lists the same ids in the same newest-first order whatever the tasks report', () => {
    const first = ids(orderTasks(report(1)))
    for (let step = 2; step <= 30; step++) {
      expect(ids(orderTasks(report(step)))).toEqual(first)
    }
    expect(first).toEqual(['walk', 'plugin-b', 'vectors', 'plugin-a'])
  })

  it('updates each task in its newest-first position with its new fields', () => {
    const before = orderTasks(report(1))
    const after = orderTasks(report(2))

    expect(after[1]?.id).toBe(before[1]?.id)
    expect(after[1]?.about).toBe('page 2')
    expect(after[2]?.id).toBe(before[2]?.id)
    expect(after[2]?.about).toBe('file 2')
  })
})

describe('calculateAggregate with a notice', () => {
  it('counts a notice as neither active nor failed', () => {
    const aggregate = calculateAggregate([
      { id: 'n', label: 'Read settings', about: 'a key is ignored', state: 'notice' },
    ])
    expect(aggregate.activeCount).toBe(0)
    expect(aggregate.failedCount).toBe(0)
    expect(aggregate.isIdle).toBe(true)
  })

  it('counts the other tasks as usual when a notice is among them', () => {
    const aggregate = calculateAggregate([
      { id: 'n', label: 'Read settings', state: 'notice' },
      { id: 'r', label: 'Indexing', progress: 30, state: 'running' },
      { id: 'f', label: 'Reading', state: 'failed', error: 'x' },
    ])
    expect(aggregate.activeCount).toBe(1)
    expect(aggregate.failedCount).toBe(1)
    expect(aggregate.overallProgress).toBe(30)
  })
})
