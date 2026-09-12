/**
 * What the window has to say, and for how long it holds on to it.
 */
import { describe, expect, it } from 'vitest'
import { messageLog } from './messages'

describe('a writer under a name', () => {
  it('holds one message at a time, and the second replaces the first', () => {
    const log = messageLog()
    const command = log.under('command')

    command('The note is in the trash')
    command('a note of that name is filed there', 'error')

    expect(log.messages.value).toHaveLength(1)
    expect(log.messages.value[0]).toMatchObject({
      name: 'command',
      kind: 'error',
      text: 'a note of that name is filed there',
    })
  })

  it('gives every message an identity of its own', () => {
    const log = messageLog()
    const command = log.under('command')

    command('One')
    const first = log.messages.value[0]?.id
    command('Two')

    expect(log.messages.value[0]?.id).not.toBe(first)
  })

  it('writes nothing where the same text already stands', () => {
    // A stream that is down says so every second, and a card that arrived
    // again is a card read out again.
    const log = messageLog()
    const worn = log.under('worn')

    worn('the themes stopped arriving', 'state')
    const first = log.messages.value[0]?.id
    worn('the themes stopped arriving', 'state')

    expect(log.messages.value[0]?.id).toBe(first)
  })

  it('clears what it wrote when it writes nothing', () => {
    const log = messageLog()
    log.under('command')('Renamed')
    log.under('made')('a note of that name is filed there', 'error')

    log.under('command')('')

    expect(log.messages.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('leaves the list alone where it had nothing to clear', () => {
    const log = messageLog()
    log.under('made')('Renamed')
    const was = log.messages.value

    log.under('command')('')

    expect(log.messages.value).toBe(was)
  })
})

describe('a message the person is finished with', () => {
  it('is dropped by the identity it was given, and nothing else is', () => {
    const log = messageLog()
    log.under('command')('Renamed')
    log.under('made')('Filed there already', 'error')
    const first = log.messages.value[0]!

    log.dismiss(first.id)

    expect(log.messages.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('is nothing at all where no message carries that identity', () => {
    const log = messageLog()
    log.under('command')('Renamed')

    log.dismiss('reading the books')

    expect(log.messages.value).toHaveLength(1)
  })
})
