import { describe, expect, it } from 'vitest'
import {
  findCloudRemoteId,
  formatVaultSize,
  getCloudRowId,
  selectNewCloudVaults,
} from './cloudVault'

const remote = (remoteId: string, vaultId: string) => ({
  remoteId,
  name: remoteId,
  sizeBytes: 1,
  vaultId,
})

describe('the size of a vault', () => {
  it.each([
    [0, '0 B'],
    [900, '900 B'],
    [1536, '1.5 KB'],
    [2040109466, '1.9 GB'],
    [5 * 1024 ** 3, '5 GB'],
    [120 * 1024 ** 2, '120 MB'],
  ])('reads %d bytes as %s', (bytes, text) => {
    expect(formatVaultSize(bytes)).toBe(text)
  })
})

describe('the cloud vaults on show', () => {
  it('leave out a vault that is connected to a vault of this device', () => {
    const shown = selectNewCloudVaults([remote('a', ''), remote('b', 'local-b')])

    expect(shown.map((one) => one.remoteId)).toStrictEqual(['a'])
  })

  it('give a row an id that no local vault has, and read the vault back from it', () => {
    expect(findCloudRemoteId(getCloudRowId('hare'))).toBe('hare')
    expect(findCloudRemoteId('3f2a-local-vault')).toBe('')
  })
})
