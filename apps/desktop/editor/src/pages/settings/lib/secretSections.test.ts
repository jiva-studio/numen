import { describe, expect, it } from 'vitest'
import { createPluginInfo } from '@/testing/installation'
import {
  compareNames,
  createSecretSections,
  getMenuSecrets,
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
    expect(compareNames('B', 'a')).toBeLessThan(0)
    expect(compareNames('ab', 'a')).toBeGreaterThan(0)
    expect(compareNames('a', 'a')).toBe(0)
  })
})

describe('the next name of a secret', () => {
  it('is the count plus one, and the first number not taken', () => {
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
