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

    expect(one.notices.value.map((said) => said.text)).toEqual(['the second', 'the third'])
    const names = one.notices.value.map((said) => said.id)
    expect(new Set(names).size).toBe(names.length)
  })

  it('says trouble in the person’s own words, and stands until they put it away', () => {
    const one = useNotices()
    one.reportError(new ConnectError('the vault could not be read', Code.Unavailable))

    const said = one.notices.value[0]!
    expect(said.text).toBe('The vault could not be read.')
    expect(said.tone).toBe('alarm')
    expect(said.stay).toBe('kept')
  })

  // Pressing a tile whose count is a moment stale is refused, and what the
  // application said is what a person needs. How it travelled is not.
  it('says a refusal without the wire it came over', () => {
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
    one.setTasks([reads()])

    expect(one.notices.value[0]).toMatchObject({
      id: 'reading\t01A',
      text: 'Reading the vault',
      about: 'Sanskrit',
      isWorking: true,
      isAsked: true,
    })
  })

  it('says why work stopped, and stands until it is put away', () => {
    const one = useNotices()
    one.setTasks([reads({ error: 'no such folder' })])

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
    one.setTasks([reads()])
    one.showNotice('the first', 'caution')
    one.setTasks([])

    expect(one.notices.value.map((said) => said.text)).toEqual(['the first'])
  })
})

/** One vault being read, as the answer holds it. */
const reads = (fields: Partial<Task> = {}): Task => ({
  id: 'reading\t01A',
  label: 'Reading the vault',
  about: 'Sanskrit',
  error: '',
  isAsked: true,
  ...fields,
})
