/**
 * The summary translates a vault into what is drawn beside it. What a vault is
 * stays the review's; the window draws what the summary returns.
 */
import { describe, expect, it } from 'vitest'
import type { VaultCardsDue } from '../../../entities/vault'
import { getVaultSummary } from './summary'
import { VAULTS_WORDS } from '../words'

const vault = (overrides: Partial<VaultCardsDue> = {}): VaultCardsDue => ({
  vault: '01A',
  name: 'Roots',
  path: '/vaults/Roots',
  isCounted: true,
  reviewCards: 0,
  due: 2,
  new: 3,
  decks: [],
  presets: [],
  unread: '',
  isIndexing: false,
  ...overrides,
})

describe('how many cards a vault has waiting', () => {
  it('is what is due and what is new, together', () => {
    expect(getVaultSummary(vault(), VAULTS_WORDS)).toStrictEqual({ kind: 'due', due: 5 })
  })

  it('is nought for a vault counted at nothing', () => {
    expect(getVaultSummary(vault({ due: 0, new: 0 }), VAULTS_WORDS)).toStrictEqual({
      kind: 'due',
      due: 0,
    })
  })

  it('is on its way for a vault not counted yet', () => {
    expect(getVaultSummary(vault({ isCounted: false }), VAULTS_WORDS)).toStrictEqual({
      kind: 'due',
      due: null,
    })
  })

  it('is on its way for a vault the count has not named yet', () => {
    expect(getVaultSummary(null, VAULTS_WORDS)).toStrictEqual({ kind: 'due', due: null })
  })
})

describe('a vault with no number to show', () => {
  it('says it is being read, in the summary own words', () => {
    expect(
      getVaultSummary(vault({ isCounted: false, isIndexing: true }), VAULTS_WORDS),
    ).toStrictEqual({ kind: 'detail', text: VAULTS_WORDS.indexing })
  })

  it('says why it could not be read, in the core words', () => {
    expect(
      getVaultSummary(vault({ isCounted: false, unread: 'the folder has gone' }), VAULTS_WORDS),
    ).toStrictEqual({ kind: 'detail', text: 'the folder has gone' })
  })
})
