/**
 * What the window has said, and for how long it holds on to it.
 */
import { describe, expect, it } from 'vitest'
import { telling } from './telling'

describe('a voice under a name', () => {
  it('says one thing at a time, and the second replaces the first', () => {
    const tell = telling()
    const command = tell.under('command')

    command('The note is in the trash')
    command('a note of that name is filed there', 'refusal')

    expect(tell.said.value).toHaveLength(1)
    expect(tell.said.value[0]).toMatchObject({
      name: 'command',
      kind: 'refusal',
      says: 'a note of that name is filed there',
    })
  })

  it('gives every utterance an identity of its own', () => {
    const tell = telling()
    const command = tell.under('command')

    command('One')
    const first = tell.said.value[0]?.id
    command('Two')

    expect(tell.said.value[0]?.id).not.toBe(first)
  })

  it('says nothing again where it is already saying it', () => {
    // A stream that is down says so every second, and a card that arrived
    // again is a card read out again.
    const tell = telling()
    const worn = tell.under('worn')

    worn('the themes stopped arriving', 'state')
    const first = tell.said.value[0]?.id
    worn('the themes stopped arriving', 'state')

    expect(tell.said.value[0]?.id).toBe(first)
  })

  it('clears what it was saying when it says nothing', () => {
    const tell = telling()
    tell.under('command')('Renamed')
    tell.under('made')('a note of that name is filed there', 'refusal')

    tell.under('command')('')

    expect(tell.said.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('leaves the list alone where it had nothing to clear', () => {
    const tell = telling()
    tell.under('made')('Renamed')
    const was = tell.said.value

    tell.under('command')('')

    expect(tell.said.value).toBe(was)
  })
})

describe('a word the person is finished with', () => {
  it('is dropped by the identity it was given, and nothing else is', () => {
    const tell = telling()
    tell.under('command')('Renamed')
    tell.under('made')('Filed there already', 'refusal')
    const first = tell.said.value[0]!

    tell.forget(first.id)

    expect(tell.said.value.map((one) => one.name)).toStrictEqual(['made'])
  })

  it('is nothing at all where no word carries that identity', () => {
    const tell = telling()
    tell.under('command')('Renamed')

    tell.forget('reading the books')

    expect(tell.said.value).toHaveLength(1)
  })
})
