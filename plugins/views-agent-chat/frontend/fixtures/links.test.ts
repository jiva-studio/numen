import { describe, expect, it } from 'vitest'
import { createLinkResolver } from './links'

describe('createLinkResolver', () => {
  it('reads the name, the heading and the alias of each wikilink', async () => {
    const resolve = createLinkResolver({ 'name://Entropy': 'notes/entropy.md' })
    expect(await resolve('See [[ Entropy#Heat#Flow |warm]], [[Order#Rank]] and [[Plain]].')).toEqual([
      { raw: '[[ Entropy#Heat#Flow |warm]]', address: 'name://Entropy', text: 'warm', path: 'notes/entropy.md' },
      { raw: '[[Order#Rank]]', address: 'name://Order', text: 'Order#Rank', path: '' },
      { raw: '[[Plain]]', address: 'name://Plain', text: 'Plain', path: '' },
    ])
  })

  it('skips a wikilink with no name', async () => {
    expect(await createLinkResolver({})('[[#Heat]] [[|alias]]')).toEqual([])
  })
})
