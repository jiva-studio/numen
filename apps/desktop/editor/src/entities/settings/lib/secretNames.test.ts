import { describe, expect, it } from 'vitest'
import {
  MAX_NAME_LENGTH,
  getMoveTarget,
  getNewGroupName,
  isAllowed,
  isGroupNameTaken,
  isNameTooLong,
  trimName,
} from './secretNames'

describe('the name of a new group', () => {
  it('is New group, then the first number after it that is not taken', () => {
    expect(getNewGroupName(['a'])).toBe('New group')
    expect(getNewGroupName(['New group'])).toBe('New group 2')
    expect(getNewGroupName(['New group', 'New group 3'])).toBe('New group 2')
    expect(getNewGroupName(['New group', 'New group 2'])).toBe('New group 3')
  })
})

describe('a name typed by the person', () => {
  it('loses U+0085 at its ends, as the core trims it', () => {
    expect(trimName('\u0085abc\u0085')).toBe('abc')
  })

  it('keeps U+FEFF at its ends, as the core does', () => {
    expect(trimName('﻿abc')).toBe('﻿abc')
  })

  it('keeps the space inside', () => {
    expect(trimName('  two words\t')).toBe('two words')
  })

  it('is too long above 64 characters and not at 64', () => {
    expect(isNameTooLong('g'.repeat(MAX_NAME_LENGTH))).toBe(false)
    expect(isNameTooLong('g'.repeat(MAX_NAME_LENGTH + 1))).toBe(true)
  })

  it('counts characters and not code units', () => {
    expect(isNameTooLong('😀'.repeat(MAX_NAME_LENGTH))).toBe(false)
  })
})

describe('the name of a group renamed', () => {
  it('is taken where another group has it, exactly', () => {
    expect(isGroupNameTaken(['alpha', 'beta'], 'alpha', 'beta')).toBe(true)
    expect(isGroupNameTaken(['alpha', 'beta'], 'alpha', 'Beta')).toBe(false)
  })

  it('is not taken by the group itself', () => {
    expect(isGroupNameTaken(['alpha', 'beta'], 'alpha', 'alpha')).toBe(false)
  })
})

describe('a permission', () => {
  it('belongs to one declaration by plugin, name and group', () => {
    const secret = {
      id: 's1',
      name: 'Work',
      group: 'openrouter',
      hasValue: true,
      allowed: [{ pluginId: 'md.numen.sample', name: 'apiKey', group: 'openrouter' }],
    }
    const key = (group: string) => ({ pluginId: 'md.numen.sample', name: 'apiKey', group })
    expect(isAllowed(secret, key('openrouter'))).toBe(true)
    expect(isAllowed(secret, key('openai'))).toBe(false)
  })
})

describe('the move of a secret', () => {
  const secrets = [{ id: 'a' }, { id: 'b' }, { id: 'c' }]

  it('goes before the one above it when it moves up', () => {
    expect(getMoveTarget(secrets, 1, 'up')).toStrictEqual({ id: 'b', beforeId: 'a' })
    expect(getMoveTarget(secrets, 2, 'up')).toStrictEqual({ id: 'c', beforeId: 'b' })
  })

  it('goes before the one two places below it when it moves down', () => {
    expect(getMoveTarget(secrets, 0, 'down')).toStrictEqual({ id: 'a', beforeId: 'c' })
  })

  it('goes to the end when it moves down from the last place but one', () => {
    expect(getMoveTarget(secrets, 1, 'down')).toStrictEqual({ id: 'b', beforeId: '' })
  })

  it('has no target above the first secret and none below the last', () => {
    expect(getMoveTarget(secrets, 0, 'up')).toBeNull()
    expect(getMoveTarget(secrets, 2, 'down')).toBeNull()
  })

  it('names no target for an index beyond the list', () => {
    expect(getMoveTarget(secrets, 5, 'up')).toBeNull()
  })
})
