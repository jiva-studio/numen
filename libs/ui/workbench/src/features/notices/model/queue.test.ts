import { describe, expect, it } from 'vitest'
import { createNoticeQueue } from './queue'

const TASK = { id: 'index', label: 'Reading the vault', about: '', error: '', isUserStarted: false }

describe('a queue of notices from several sources', () => {
  it('lists the messages of every source oldest first, each with a unique id', () => {
    const queue = createNoticeQueue()
    queue.show('sample', 'The sample was saved.', 'info')
    queue.show('graph', 'The note is gone.', 'warning')
    queue.show('sample', 'The sample was saved.', 'info')

    const ids = queue.notices.value.map((one) => one.id)
    expect(ids).toEqual(['sample#1', 'graph#2', 'sample#3'])
    expect(queue.notices.value.map((one) => one.text)).toEqual([
      'The sample was saved.',
      'The note is gone.',
      'The sample was saved.',
    ])
  })

  it('lets an info notice go once read and keeps any other until it is dismissed', () => {
    const queue = createNoticeQueue()
    queue.show('sample', 'Saved.', 'info')
    queue.show('sample', 'Could not save.', 'error')

    expect(queue.notices.value.map((one) => [one.severity, one.stay])).toEqual([
      ['info', 'read'],
      ['error', 'kept'],
    ])
    expect(queue.notices.value.every((one) => one.isUserStarted && !one.isWorking)).toBe(true)
  })

  it('says nothing for an empty text', () => {
    const queue = createNoticeQueue()
    queue.show('sample', '', 'error')
    expect(queue.notices.value).toEqual([])
  })

  it('lists the tasks of every source before the messages', () => {
    const queue = createNoticeQueue()
    queue.show('sample', 'Saved.', 'info')
    queue.setTasks('sample', [TASK])

    expect(queue.notices.value.map((one) => one.text)).toEqual(['Reading the vault', 'Saved.'])
    expect(queue.notices.value[0]?.isWorking).toBe(true)
    expect(queue.notices.value[0]?.isUserStarted).toBe(false)
    expect(queue.notices.value[0]?.severity).toBeUndefined()
    expect(queue.notices.value[0]?.done).toBeUndefined()
    expect(queue.notices.value[0]?.total).toBeUndefined()
  })

  it('turns a failed or counted task into a notice with its error and progress', () => {
    const queue = createNoticeQueue()
    queue.setTasks('sample', [
      { ...TASK, id: 'one', error: 'Disk full', isUserStarted: false },
      { ...TASK, id: 'two', isUserStarted: true, done: 3, total: 10, counting: 'files' as never },
      { ...TASK, id: 'three', done: 2, total: 5 },
      { ...TASK, id: 'four', done: 0, total: 0 },
      { ...TASK, id: 'five', done: 1 },
      { ...TASK, id: 'six', total: 4 },
    ])

    const [failed, counted, plainCount, zeroTotal, noTotal, noDone] = queue.notices.value
    expect(failed).toMatchObject({
      id: 'sample:one',
      text: 'Disk full',
      isWorking: false,
      isUserStarted: true,
      severity: 'error',
      stay: 'kept',
    })
    expect(counted).toMatchObject({
      id: 'sample:two',
      isWorking: true,
      isUserStarted: true,
      done: 3,
      total: 10,
      counting: 'files',
    })
    expect(plainCount).toMatchObject({ id: 'sample:three', done: 2, total: 5 })
    expect('counting' in plainCount!).toBe(false)
    expect(zeroTotal?.done).toBeUndefined()
    expect(zeroTotal?.total).toBeUndefined()
    expect(noTotal?.done).toBeUndefined()
    expect(noDone?.total).toBeUndefined()
  })

  it('replaces only the work of the source that gives a new list', () => {
    const queue = createNoticeQueue()
    queue.setTasks('sample', [TASK])
    queue.setTasks('graph', [{ ...TASK, label: 'Drawing' }])
    queue.setTasks('sample', [])

    expect(queue.notices.value.map((one) => one.text)).toEqual(['Drawing'])
  })

  it('keeps two sources apart when their tasks share an identity', () => {
    const queue = createNoticeQueue()
    queue.setTasks('sample', [TASK])
    queue.setTasks('graph', [TASK])

    const ids = queue.notices.value.map((one) => one.id)
    expect(ids).toEqual(['sample:index', 'graph:index'])
  })

  it('takes away one notice when it is dismissed', () => {
    const queue = createNoticeQueue()
    queue.show('sample', 'One.', 'warning')
    queue.show('sample', 'Two.', 'warning')

    queue.dismiss(queue.notices.value[0]?.id ?? '')

    expect(queue.notices.value.map((one) => one.text)).toEqual(['Two.'])
  })

  it('takes away everything a source gave when it is cleared', () => {
    const queue = createNoticeQueue()
    queue.show('sample', 'Saved.', 'info')
    queue.setTasks('sample', [TASK])
    queue.show('graph', 'Drawn.', 'info')

    queue.clear('sample')

    expect(queue.notices.value.map((one) => one.text)).toEqual(['Drawn.'])
  })
})
