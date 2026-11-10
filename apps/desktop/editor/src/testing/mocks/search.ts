/**
 * The search part of the vault mock: the note names and passages a query finds.
 */
import { mockVaultState } from '../answers'
import type { VaultCore } from '@/app/vault/core'

export const searchCoreMock = {
  names: () => Promise.resolve(mockVaultState.names),
  search: () => Promise.resolve(mockVaultState.passages),
} satisfies Partial<VaultCore>
