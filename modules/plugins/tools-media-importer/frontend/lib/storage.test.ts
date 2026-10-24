/**
 * The height a person drew the player, kept between runs.
 */
import { describe, expect, it } from 'vitest'
import { LEAST } from './height'
import { createHeightStore, type KeyValueStorage } from './storage'

const createMemory = (): KeyValueStorage & { held: Map<string, string> } => {
  const target = new Map<string, string>()
  return {
    held: target,
    getItem: (key) => target.get(key) ?? null,
    setItem: (key, value) => void target.set(key, value),
  }
}

const refusing: KeyValueStorage = {
  getItem: () => {
    throw new Error('denied')
  },
  setItem: () => {
    throw new Error('denied')
  },
}

describe('the height kept', () => {
  it('is given back as it was written', () => {
    const memory = createMemory()
    const store = createHeightStore(() => memory)
    store.write(300)
    expect(store.read()).toBe(300)
  })

  it('is nothing where none was written', () => {
    expect(createHeightStore(() => createMemory()).read()).toBeNull()
  })

  it('is nothing where it is shorter than the player is drawn', () => {
    const memory = createMemory()
    const store = createHeightStore(() => memory)
    store.write(LEAST - 1)
    expect(store.read()).toBeNull()
  })

  it('is nothing, and loses nothing else, where the machine keeps nothing', () => {
    const store = createHeightStore(() => refusing)
    expect(() => store.write(300)).not.toThrow()
    expect(store.read()).toBeNull()
  })
})
