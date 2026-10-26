import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { createMockNoticeHost } from '@numen/plugin-sdk/testing'

import { createNotices } from './notices'

const createWindowNotices = () => {
  const host = createMockNoticeHost()
  return { host, ...createNotices(host) }
}

describe('what the window has to say', () => {
  it('shows each notice in its tone', () => {
    const one = createWindowNotices()
    one.showNotice('the first', 'caution')
    one.showNotice('the second', 'alarm')

    expect(one.host.getShown()).toEqual([
      { text: 'the first', tone: 'caution' },
      { text: 'the second', tone: 'alarm' },
    ])
  })

  it('says trouble in the person’s own words, as an alarm', () => {
    const one = createWindowNotices()
    one.reportError(new ConnectError('Vault unreadable', Code.Unavailable))

    expect(one.host.getShown()).toEqual([{ text: 'Vault unreadable.', tone: 'alarm' }])
  })

  // Pressing a tile whose count is a moment stale fails, and what the
  // application said is what a person needs. How it travelled is not.
  it('says an error without the wire it came over', () => {
    const one = createWindowNotices()
    one.reportError(
      new ConnectError(
        'this preset schedules nothing today: it is paused',
        Code.FailedPrecondition,
      ),
    )

    expect(one.host.getShown()[0]!.text).toBe('This preset schedules nothing today: it is paused.')
  })

  it('leaves a sentence that already ends where it ends', () => {
    const one = createWindowNotices()
    one.reportError(new ConnectError('The deck could not be written.', Code.Unavailable))

    expect(one.host.getShown()[0]!.text).toBe('The deck could not be written.')
  })

  it('ends a sentence whose only stop remains inside it', () => {
    const one = createWindowNotices()
    one.reportError(
      new ConnectError('is the preset paused? failed to open', Code.FailedPrecondition),
    )

    expect(one.host.getShown()[0]!.text).toBe('Is the preset paused? failed to open.')
  })

  it('raises nothing for a call the window itself stopped', () => {
    const one = createWindowNotices()
    one.reportError(new ConnectError('the window closed', Code.Canceled))

    expect(one.host.getShown()).toHaveLength(0)
  })

  it('sets the whole task list at once', () => {
    const one = createWindowNotices()
    const task = { id: 'indexing\t01A', label: 'Indexing the vault', about: 'Sanskrit' }
    one.setTasks([task])

    expect(one.host.getTasks()).toEqual([task])
  })
})
