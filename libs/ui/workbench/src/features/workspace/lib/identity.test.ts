/**
 * Identities are a port, so a test holds them still.
 *
 * A machine with no randomness of its own still lays out a workspace, and the
 * names it gives are its own count and not the clock.
 */
import { describe, expect, it } from 'vitest'
import { createNodeIdFactory } from './identity'

describe('the identities a workspace gives its nodes', () => {
  it('are what the randomness passes over', () => {
    const createId = createNodeIdFactory({ uuid: () => 'from-the-machine' })

    expect(createId()).toBe('from-the-machine')
  })

  it('are counted where the machine makes none', () => {
    const createId = createNodeIdFactory({ uuid: () => null })

    expect([createId(), createId(), createId()]).toStrictEqual(['node-1', 'node-2', 'node-3'])
  })

  it('are counted apart, so two factories never meet', () => {
    const one = createNodeIdFactory({ uuid: () => null })
    const other = createNodeIdFactory({ uuid: () => null })

    expect(one()).toBe(other())
    expect(one()).not.toBe(other() + 'x')
  })
})
