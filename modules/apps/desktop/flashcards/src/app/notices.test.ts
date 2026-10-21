import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'

import { useNotices } from './notices'
import type { Task } from '@numen/ui'

describe('what the window has to say', () => {
  it('names each thing once, so putting one away leaves the rest', () => {
    const one = useNotices()
    one.showNotice('the first', 'caution')
    one.showNotice('the second', 'caution')

    const first = one.notices.value[0]!
    one.dismissNotice(first.id)
    one.showNotice('the third', 'caution')

    expect(one.notices.value.map((saidText) => saidText.text)).toEqual(['the second', 'the third'])
    const names = one.notices.value.map((response) => response.id)
    expect(new Set(names).size).toBe(names.length)
  })

  it('numbers what it says in the order it was returned', () => {
    const one = useNotices()
    one.showNotice('the first', 'caution')
    one.showNotice('the second', 'caution')

    expect(one.notices.value.map((response) => response.id)).toEqual(['1', '2'])
  })

  it('says trouble in the person’s own words, and remains until they put it away', () => {
    const one = useNotices()
    one.reportError(new ConnectError('the vault could not be read', Code.Unavailable))

    const response = one.notices.value[0]!
    expect(response.text).toBe('The vault could not be read.')
    expect(response.tone).toBe('alarm')
    expect(response.stay).toBe('kept')
  })

  // Pressing a tile whose count is a moment stale fails, and what the
  // application said is what a person needs. How it travelled is not.
  it('says an error without the wire it came over', () => {
    const one = useNotices()
    one.reportError(
      new ConnectError(
        'this preset schedules nothing today: it is paused',
        Code.FailedPrecondition,
      ),
    )

    expect(one.notices.value[0]!.text).toBe('This preset schedules nothing today: it is paused.')
  })

  it('leaves a sentence that already ends where it ends', () => {
    const one = useNotices()
    one.reportError(new ConnectError('The deck could not be written.', Code.Unavailable))

    expect(one.notices.value[0]!.text).toBe('The deck could not be written.')
  })

  it('ends a sentence whose only stop remains inside it', () => {
    const one = useNotices()
    one.reportError(
      new ConnectError('is the preset paused? the deck says so', Code.FailedPrecondition),
    )

    expect(one.notices.value[0]!.text).toBe('Is the preset paused? the deck says so.')
  })

  it('raises nothing for a call the window itself stopped', () => {
    const one = useNotices()
    one.reportError(new ConnectError('the window closed', Code.Canceled))

    expect(one.notices.value).toHaveLength(0)
  })

  it('draws what it was told at once, a person being there waiting to hear it', () => {
    const one = useNotices()
    one.showNotice('the first', 'caution')

    expect(one.notices.value[0]).toMatchObject({ tone: 'caution', isUserStarted: true })
  })

  it('puts away nothing when the name is not one it holds', () => {
    const one = useNotices()
    one.showNotice('the first', 'caution')
    one.dismissNotice('nothing')

    expect(one.notices.value).toHaveLength(1)
  })

  // Reading a vault is what this window does behind itself, and a person opened
  // the window on that vault, so the card is drawn the moment it arrives.
  it('draws work being done, and draws it at once', () => {
    const one = useNotices()
    one.setTasks([createTask()])

    expect(one.notices.value[0]).toMatchObject({
      id: 'indexing\t01A',
      text: 'Indexing the vault',
      about: 'Sanskrit',
      isWorking: true,
      isUserStarted: true,
    })
  })

  it('says why work stopped, and remains until it is put away', () => {
    const one = useNotices()
    one.setTasks([createTask({ error: 'no such folder' })])

    expect(one.notices.value[0]).toMatchObject({
      text: 'no such folder',
      isWorking: false,
      tone: 'alarm',
      stay: 'kept',
    })
  })

  // The whole list arrives at once, so the whole list is what stands: work that
  // finished is work the next list leaves out.
  it('lets go of work the list no longer names', () => {
    const one = useNotices()
    one.setTasks([createTask()])
    one.showNotice('the first', 'caution')
    one.setTasks([])

    expect(one.notices.value.map((notice) => notice.text)).toEqual(['the first'])
  })
})

/** Creates a task fixture for testing. */
const createTask = (fields: Partial<Task> = {}): Task => ({
  id: 'indexing\t01A',
  label: 'Indexing the vault',
  about: 'Sanskrit',
  error: '',
  isUserStarted: true,
  ...fields,
})
