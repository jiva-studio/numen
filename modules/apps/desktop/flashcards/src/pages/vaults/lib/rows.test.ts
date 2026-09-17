/**
 * The screen translates a vault into a row. What a vault is stays this
 * window's, and the component is told only what to draw.
 */
import { describe, expect, it } from 'vitest'
import type { VaultCardsDue } from '@/entities/vault'
import { getDueByVault, getVaultRows } from './rows'
import { VAULTS_WORDS } from '../words'

const vault = (over: Partial<VaultCardsDue> = {}): VaultCardsDue => ({
  vault: '01A',
  name: 'Roots',
  path: '/vaults/Roots',
  counted: true,
  faces: 0,
  due: 2,
  new: 3,
  decks: [],
  presets: [],
  unread: '',
  reading: false,
  ...over,
})

describe('a vault as a row', () => {
  it('carries no detail where there is nothing to say', () => {
    expect(getVaultRows([vault()], VAULTS_WORDS)[0]).toStrictEqual({
      id: '01A',
      name: 'Roots',
      path: '/vaults/Roots',
      isWorking: false,
    })
  })

  it('says it is being read, in the screen own words', () => {
    const row = getVaultRows([vault({ counted: false, reading: true })], VAULTS_WORDS)[0]

    expect(row?.detail).toBe(VAULTS_WORDS.reading)
    expect(row?.isWorking).toBe(true)
  })

  it('says why it could not be read, in the core words', () => {
    const row = getVaultRows([vault({ counted: false, unread: 'the folder has gone' })], VAULTS_WORDS)[0]

    expect(row?.detail).toBe('the folder has gone')
  })
})

describe('how many cards a vault has waiting', () => {
  it('is what is due and what is new, together', () => {
    expect(getDueByVault([vault()]).get('01A')).toBe(5)
  })

  it('is nothing for a vault counted to nothing', () => {
    expect(getDueByVault([vault({ counted: false })]).get('01A')).toBeNull()
  })

  it('is absent for a vault being read, which says why in its row instead', () => {
    expect(getDueByVault([vault({ reading: true })]).has('01A')).toBe(false)
  })

  it('is absent for a vault that could not be read', () => {
    expect(getDueByVault([vault({ unread: 'the folder has gone' })]).has('01A')).toBe(false)
  })
})
