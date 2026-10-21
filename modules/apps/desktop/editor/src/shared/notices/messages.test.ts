/**
 * What the window has to say, and for how long it holds on to it.
 */
import { describe, expect, it } from 'vitest'
import { createMessageLog } from './messages'

describe('a writer under a name', () => {
  it('holds one message at a time, and the second replaces the first', () => {
    const log = createMessageLog()
    const command = log.getWriter('command')

    command('The note is in the trash')
    command('a note of that name is filed there', 'error')

    expect(log.messages.value).toHaveLength(1)
    expect(log.messages.value[0]).toMatchObject({
      name: 'command',
      kind: 'error',
      text: 'a note of that name is filed there',
    })
  })

  it('writes a report where no kind is given', () => {
    const log = createMessageLog()
    log.getWriter('command')('Renamed')

    expect(log.messages.value[0]?.kind).toBe('report')
  })

  it('replaces what remains where only its kind differs', () => {
    const log = createMessageLog()
    const command = log.getWriter('command')

    command('Renamed', 'report')
    command('Renamed', 'error')

    expect(log.messages.value).toHaveLength(1)
    expect(log.messages.value[0]?.kind).toBe('error')
  })

  it('numbers the identities upward from one, across every name', () => {
    const log = createMessageLog()
    log.getWriter('command')('One')
    log.getWriter('made')('Two')
    log.getWriter('command')('Three')

    expect(log.messages.value.map((one) => one.id)).toStrictEqual(['made#2', 'command#3'])
  })

  it('gives every message an identity of its own', () => {
    const log = createMessageLog()
    const command = log.getWriter('command')

    command('One')
    const first = log.messages.value[0]?.id
    command('Two')

    expect(log.messages.value[0]?.id).not.toBe(first)
  })

  it('writes nothing where the same text already remains', () => {
    // A stream that is down says so every second, and a card that arrived
    // again is a card read out again.
    const log = createMessageLog()
    const worn = log.getWriter('worn')

    worn('the themes stopped arriving', 'state')
    const first = log.messages.value[0]?.id
    worn('the themes stopped arriving', 'state')

    expect(log.messages.value[0]?.id).toBe(first)
  })

  it('clears what it wrote when it writes nothing', () => {
    const log = createMessageLog()
    log.getWriter('command')('Renamed')
    log.getWriter('made')('a note of that name is filed there', 'error')

    log.getWriter('command')('')

    expect(log.messages.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('leaves the list alone where it had nothing to clear', () => {
    const log = createMessageLog()
    log.getWriter('made')('Renamed')
    const was = log.messages.value

    log.getWriter('command')('')

    expect(log.messages.value).toBe(was)
  })
})

describe('a message the person is finished with', () => {
  it('is dropped by the identity it was given, and nothing else is', () => {
    const log = createMessageLog()
    log.getWriter('command')('Renamed')
    log.getWriter('made')('Filed there already', 'error')
    const first = log.messages.value[0]!

    log.dismiss(first.id)

    expect(log.messages.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('is nothing at all where no message carries that identity', () => {
    const log = createMessageLog()
    log.getWriter('command')('Renamed')

    log.dismiss('extracting the books')

    expect(log.messages.value).toHaveLength(1)
  })
})
