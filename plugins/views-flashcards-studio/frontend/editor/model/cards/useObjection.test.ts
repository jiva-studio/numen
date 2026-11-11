import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { HeadingObjection } from '../../../lib/cards'
import type { NameEntryState } from './nameEntry'
import { useObjection, useObjectionMessage } from './useObjection'

type Why = HeadingObjection

const createNaming = (objections: Readonly<Record<string, Why>>): NameEntryState<Why> => ({
  getText: () => '',
  getObjection: (targetId) => objections[targetId] ?? null,
  setDraft: () => undefined,
  commit: () => undefined,
  onKey: () => undefined,
})

describe('why a name cannot be used', () => {
  it('is the objection the name entry holds for the target', () => {
    const naming = createNaming({ front: 'taken' })

    expect(
      useObjection(
        () => naming,
        () => 'front',
      ).value,
    ).toBe('taken')
  })

  it('is null for a target without one', () => {
    const naming = createNaming({ front: 'taken' })

    expect(
      useObjection(
        () => naming,
        () => 'back',
      ).value,
    ).toBeNull()
  })

  it('follows a change of the target id', () => {
    const naming = createNaming({ front: 'taken' })
    const target = ref('back')
    const objection = useObjection(
      () => naming,
      () => target.value,
    )

    target.value = 'front'

    expect(objection.value).toBe('taken')
  })
})

describe('the message for an objection', () => {
  const format = (why: Why): string => (why === 'blank' ? 'Say a name' : 'Already used')

  it('is the formatted objection', () => {
    expect(useObjectionMessage({ value: 'taken' }, () => format).value).toBe('Already used')
  })

  it('is null while there is no objection', () => {
    expect(useObjectionMessage({ value: null }, () => format).value).toBeNull()
  })
})
