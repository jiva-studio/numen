/**
 * The vault mock the window uses, responding from `vaultState`. An entity that calls the core
 * through its own `api` is answered through the router transport, from `mocks/<entity>.ts`; the
 * search is answered through the mocked `core` object.
 */
import { vi } from 'vitest'
import './mocks/plugin'
import { searchCoreMock } from './mocks/search'
import type { VaultCore } from '@/app/vault/core'

vi.mock('@/app/vault', () => ({
  core: {
    ...searchCoreMock,
  } satisfies Partial<VaultCore>,
}))
