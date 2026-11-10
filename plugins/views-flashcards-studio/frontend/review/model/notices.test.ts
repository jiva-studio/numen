import { describe, expect, it } from 'vitest'
import { createMockNoticeHost } from '@numen/plugin-sdk/testing'

import { ApiFailure, CANCELED } from '../../api/failure'
import { createNotices } from './notices'

const createWindowNotices = () => {
  const host = createMockNoticeHost()
  return { host, ...createNotices(host) }
}

describe('what the window has to say', () => {
  it('shows each notice in its severity', () => {
    const one = createWindowNotices()
    one.showNotice('the first', 'warning')
    one.showNotice('the second', 'error')

    expect(one.host.getShown()).toEqual([
      { text: 'the first', severity: 'warning' },
      { text: 'the second', severity: 'error' },
    ])
  })

  it('says trouble in the person’s own words, as an error', () => {
    const one = createWindowNotices()
    one.reportError(new ApiFailure({ code: 'unavailable', message: 'Vault unreadable' }))

    expect(one.host.getShown()).toEqual([{ text: 'Vault unreadable.', severity: 'error' }])
  })

  // Pressing a tile whose count is out of date fails, and the user needs the
  // application's message. The transport is irrelevant to them.
  it('says an error without the wire it came over', () => {
    const one = createWindowNotices()
    one.reportError(
      new ApiFailure({
        code: 'failed_precondition',
        message: 'this preset schedules nothing today: it is paused',
      }),
    )

    expect(one.host.getShown()[0]!.text).toBe('This preset schedules nothing today: it is paused.')
  })

  it('leaves a sentence that already ends where it ends', () => {
    const one = createWindowNotices()
    one.reportError(
      new ApiFailure({ code: 'unavailable', message: 'The deck could not be written.' }),
    )

    expect(one.host.getShown()[0]!.text).toBe('The deck could not be written.')
  })

  it('ends a sentence whose only stop remains inside it', () => {
    const one = createWindowNotices()
    one.reportError(
      new ApiFailure({
        code: 'failed_precondition',
        message: 'is the preset paused? failed to open',
      }),
    )

    expect(one.host.getShown()[0]!.text).toBe('Is the preset paused? failed to open.')
  })

  it('raises nothing for a call the window itself stopped', () => {
    const one = createWindowNotices()
    one.reportError(new ApiFailure(CANCELED))

    expect(one.host.getShown()).toHaveLength(0)
  })

  it('sets the whole task list at once', () => {
    const one = createWindowNotices()
    const task = { id: 'indexing\t01A', label: 'Indexing the vault', about: 'Sanskrit' }
    one.setTasks([task])

    expect(one.host.getTasks()).toEqual([task])
  })
})
