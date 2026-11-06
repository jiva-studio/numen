import { describe, expect, it } from 'vitest'
import { createTask } from '../testing/tasks'
import { getProgress, orderTasks, summarizeTasks } from './summary'

describe('summarizeTasks', () => {
  it('is idle when no task runs and none failed', () => {
    const summary = summarizeTasks([createTask({ id: 'n1', state: 'notice' })])

    expect(summary).toStrictEqual({
      activeCount: 0,
      failedCount: 0,
      progress: undefined,
      isIdle: true,
    })
  })

  it('counts running and cancelling tasks as active, and failed tasks apart', () => {
    const summary = summarizeTasks([
      createTask({ id: 'a', state: 'running' }),
      createTask({ id: 'b', state: 'cancelling' }),
      createTask({ id: 'c', state: 'failed', error: 'disk full' }),
      createTask({ id: 'd', state: 'notice' }),
    ])

    expect(summary.activeCount).toBe(2)
    expect(summary.failedCount).toBe(1)
    expect(summary.isIdle).toBe(false)
  })

  it('is not idle while only failed tasks are listed', () => {
    const summary = summarizeTasks([createTask({ id: 'c', state: 'failed', error: 'disk full' })])

    expect(summary.isIdle).toBe(false)
  })

  it('averages the progress of the active tasks that have a total', () => {
    const summary = summarizeTasks([
      createTask({ id: 'a', state: 'running', done: 1, total: 4 }),
      createTask({ id: 'b', state: 'running', done: 3, total: 4 }),
      createTask({ id: 'c', state: 'running' }),
      createTask({ id: 'd', state: 'failed', done: 0, total: 10, error: 'stopped' }),
    ])

    expect(summary.progress).toBe(50)
  })

  it('has no progress while no active task has a total', () => {
    const summary = summarizeTasks([createTask({ id: 'a', state: 'running' })])

    expect(summary.progress).toBeUndefined()
  })
})

describe('getProgress', () => {
  it('is the rounded percent of the total that is done', () => {
    expect(getProgress(createTask({ id: 'a', done: 1, total: 3 }))).toBe(33)
  })

  it('is undefined for a task without a total', () => {
    expect(getProgress(createTask({ id: 'a', done: 0, total: 0 }))).toBeUndefined()
  })
})

describe('orderTasks', () => {
  it('puts the newest task first', () => {
    const tasks = [createTask({ id: 'old' }), createTask({ id: 'mid' }), createTask({ id: 'new' })]

    expect(orderTasks(tasks).map((one) => one.id)).toStrictEqual(['new', 'mid', 'old'])
  })
})
