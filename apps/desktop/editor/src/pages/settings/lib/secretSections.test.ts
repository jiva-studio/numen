import { describe, expect, it } from 'vitest'
import { createPluginInfo } from '@/testing/installation'
import {
  createSecretSections,
  getMenuSecrets,
  getNewGroupName,
  getSectionDeclarations,
  getDeclarations,
  getNextSecretName,
  getReaderNames,
  hasGroupSecrets,
  isAllowed,
  type Declaration,
} from './secretSections'

const plugin = createPluginInfo('md.numen.sample', 'Sample', {
  secrets: [
    { name: 'apiKey', title: 'Key', description: 'A key', group: 'openrouter' },
    { name: 'apiKey', title: 'Key', description: 'Another key', group: 'openai' },
    { name: 'token', title: 'Token', description: 'A token', group: '' },
  ],
})

const secret = (id: string, name: string, group: string) => ({
  id,
  name,
  group,
  allowed: [],
  hasValue: true,
})

describe('the sections of the secrets', () => {
  it('lists plugin groups by name, then person groups by name, and keeps a plugin group with no secret', () => {
    const sections = createSecretSections(
      [plugin],
      [secret('s1', 'a', 'zeta'), secret('s2', 'b', 'alpha'), secret('s3', 'c', 'openrouter')],
    )
    expect(sections.map((one) => [one.group, one.isAsked, one.secrets.length])).toStrictEqual([
      ['openai', true, 0],
      ['openrouter', true, 1],
      ['alpha', false, 1],
      ['zeta', false, 1],
    ])
  })

  it('order names by their bytes, uppercase before lowercase', () => {
    const sections = createSecretSections(
      [],
      [secret('s1', 'a', 'a'), secret('s2', 'b', 'B'), secret('s3', 'c', 'ab')],
    )
    expect(sections.map((one) => one.group)).toStrictEqual(['B', 'a', 'ab'])
  })
})

describe('the next name of a secret', () => {
  it('is the count plus one, and the first number not taken', () => {
    expect(getNextSecretName([])).toBe('Default')
    expect(getNextSecretName(['Default'])).toBe('2')
    expect(getNextSecretName(['Work', 'Home'])).toBe('3')
    expect(getNextSecretName(['1', '3'])).toBe('4')
    expect(getNextSecretName(['2'])).toBe('3')
  })
})

describe('the menu of a declaration', () => {
  const sections = createSecretSections(
    [plugin],
    [secret('s1', 'Work', 'openrouter'), secret('s2', 'Chat', 'openai'), secret('s3', 'Mine', 'x')],
  )
  const declaration = (group: string): Declaration => ({
    pluginId: 'md.numen.sample',
    pluginName: 'Sample',
    name: 'apiKey',
    group,
    title: 'Key',
    description: '',
  })

  it('puts the secrets of the declared group first', () => {
    expect(
      getMenuSecrets(sections, declaration('openrouter')).map((one) => one.name),
    ).toStrictEqual(['Work', 'Chat', 'Mine'])
  })

  it('keeps the order of the page for a declaration with no group', () => {
    expect(getMenuSecrets(sections, declaration('')).map((one) => one.name)).toStrictEqual([
      'Chat',
      'Work',
      'Mine',
    ])
  })

  it('reports whether the declared group has a secret', () => {
    expect(hasGroupSecrets(sections, declaration('openai'))).toBe(true)
    expect(hasGroupSecrets(sections, declaration('mistral'))).toBe(false)
    expect(hasGroupSecrets(sections, declaration(''))).toBe(false)
  })
})

describe('a permission', () => {
  it('belongs to one declaration by plugin, name and group', () => {
    const allowed = {
      ...secret('s1', 'Work', 'openrouter'),
      allowed: [{ pluginId: 'md.numen.sample', name: 'apiKey', group: 'openrouter' }],
    }
    const declaration = (group: string): Declaration => ({
      pluginId: 'md.numen.sample',
      pluginName: 'Sample',
      name: 'apiKey',
      group,
      title: '',
      description: '',
    })
    expect(isAllowed(allowed, declaration('openrouter'))).toBe(true)
    expect(isAllowed(allowed, declaration('openai'))).toBe(false)
    expect(getReaderNames(allowed, [plugin])).toStrictEqual(['Sample'])
    expect(getReaderNames(allowed, [])).toStrictEqual(['md.numen.sample'])
  })
})

describe('the declarations of a section', () => {
  const all = getDeclarations([plugin])

  it('are those of its group and those with no group', () => {
    expect(getSectionDeclarations(all, 'openrouter').map((one) => one.description)).toStrictEqual([
      'A key',
      'A token',
    ])
    expect(getSectionDeclarations(all, 'openai').map((one) => one.description)).toStrictEqual([
      'Another key',
      'A token',
    ])
  })

  it('are only those with no group in a group no plugin asks for', () => {
    expect(getSectionDeclarations(all, 'travel').map((one) => one.name)).toStrictEqual(['token'])
  })

  it('list a declaration once', () => {
    expect(getSectionDeclarations([...all, ...all], 'openai')).toHaveLength(2)
  })
})

describe('the name of a new group', () => {
  it('is New group, then the first number after it that is not taken', () => {
    expect(getNewGroupName(['a'])).toBe('New group')
    expect(getNewGroupName(['New group'])).toBe('New group 2')
    expect(getNewGroupName(['New group', 'New group 3'])).toBe('New group 2')
    expect(getNewGroupName(['New group', 'New group 2'])).toBe('New group 3')
  })
})

describe('the sections with groups the person made and left empty', () => {
  it('list them with the groups of the secrets, once', () => {
    const sections = createSecretSections([], [secret('s1', 'a', 'beta')], ['alpha', 'beta'])
    expect(sections.map((one) => [one.group, one.secrets.length])).toStrictEqual([
      ['alpha', 0],
      ['beta', 1],
    ])
  })
})
