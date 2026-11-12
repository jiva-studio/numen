import { describe, expect, it } from 'vitest'
import { createPluginInfo, createSecretAnswer } from '@/pages/settings/testing/installation'
import { WORDS } from '../words'
import {
  createPageSections,
  getMenuSecrets,
  getSectionDeclarations,
  getDeclarations,
  getDeletionText,
  hasGroupSecrets,
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

const WORK = secret('s1', 'Work', 'openrouter')
const CHAT = secret('s2', 'Chat', 'openai')

describe('the sections of the page', () => {
  const core = createSecretAnswer(
    [plugin],
    [secret('s1', 'a', 'zeta'), secret('s2', 'b', 'alpha'), secret('s3', 'c', 'openrouter')],
  )

  it('keep the order the core gave and add the groups of the window after them', () => {
    const sections = createPageSections(core.sections?.value ?? [], ['mine'], new Map())
    expect(sections.map((one) => [one.group, one.isAsked, one.secrets.length])).toStrictEqual([
      ['openai', true, 0],
      ['openrouter', true, 1],
      ['alpha', false, 1],
      ['zeta', false, 1],
      ['mine', false, 0],
    ])
  })

  it('get the state of each value, and none for a secret with no state', () => {
    const sections = createPageSections(core.sections?.value ?? [], [], new Map([['s1', true]]))
    const secrets = sections.flatMap((one) => one.secrets)
    expect(secrets.map((one) => [one.id, one.hasValue])).toStrictEqual([
      ['s3', false],
      ['s2', false],
      ['s1', true],
    ])
  })

  it('list a group of the window once, and not where the core has it', () => {
    const sections = createPageSections(
      core.sections?.value ?? [],
      ['alpha', 'new', 'new'],
      new Map(),
    )
    expect(sections.map((one) => one.group)).toStrictEqual([
      'openai',
      'openrouter',
      'alpha',
      'zeta',
      'new',
    ])
  })
})

describe('the menu of a declaration', () => {
  const answer = createSecretAnswer(
    [plugin],
    [secret('s1', 'Work', 'openrouter'), secret('s2', 'Chat', 'openai'), secret('s3', 'Mine', 'x')],
  )
  const sections = answer.sections?.value ?? []
  const menus = [
    { pluginId: 'md.numen.sample', name: 'apiKey', group: 'openrouter', secrets: [WORK, CHAT] },
  ]
  const declaration = (group: string): Declaration => ({
    pluginId: 'md.numen.sample',
    pluginName: 'Sample',
    name: 'apiKey',
    group,
    title: 'Key',
    description: '',
  })

  it('is the menu the core gave for the plugin, name and group, in its order', () => {
    expect(getMenuSecrets(menus, declaration('openrouter')).map((one) => one.name)).toStrictEqual([
      'Work',
      'Chat',
    ])
  })

  it('is empty for a declaration the core gave no menu for', () => {
    expect(getMenuSecrets(menus, declaration(''))).toStrictEqual([])
  })

  it('reports whether the declared group has a secret', () => {
    expect(hasGroupSecrets(sections, declaration('openai'))).toBe(true)
    expect(hasGroupSecrets(sections, declaration('mistral'))).toBe(false)
    expect(hasGroupSecrets(sections, declaration(''))).toBe(false)
  })
})

describe('the readers of a secret', () => {
  it('are the plugins allowed on it, each by name, else by id', () => {
    const allowed = {
      ...secret('s1', 'Work', 'openrouter'),
      allowed: [{ pluginId: 'md.numen.sample', name: 'apiKey', group: 'openrouter' }],
    }
    const target = { kind: 'secret', secret: allowed } as const
    expect(getDeletionText(target, [plugin])).toBe(WORDS.secretDeleteAsking('Work', ['Sample']))
    expect(getDeletionText(target, [])).toBe(WORDS.secretDeleteAsking('Work', ['md.numen.sample']))
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

  it('list a plugin once for a name: the ungrouped one is left out where the plugin declares the name for the group', () => {
    const both = getDeclarations([
      createPluginInfo('md.numen.sample', 'Sample', {
        secrets: [
          { name: 'apiKey', title: 'Key', description: 'Of the group', group: 'openai' },
          { name: 'apiKey', title: 'Key', description: 'Of no group', group: '' },
        ],
      }),
    ])
    expect(getSectionDeclarations(both, 'openai').map((one) => one.description)).toStrictEqual([
      'Of the group',
    ])
    expect(getSectionDeclarations(both, 'travel').map((one) => one.description)).toStrictEqual([
      'Of no group',
    ])
  })

  it('list the declarations of a group in the order they are declared', () => {
    expect(
      getSectionDeclarations([...all].reverse(), 'openai').map((one) => one.description),
    ).toStrictEqual(['A token', 'Another key'])
  })
})
