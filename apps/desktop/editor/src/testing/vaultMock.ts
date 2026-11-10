/**
 * The vault mock the window uses, responding from `vaultState` and recording requests
 * in `requests`. An entity that calls the core through its own `api` is answered through the
 * router transport, from `mocks/<entity>.ts`; the rest is answered through the mocked `core` object.
 */
import { vi } from 'vitest'
import './mocks/plugin'
import { searchCoreMock } from './mocks/search'
import { tabCoreMock } from './mocks/tab'
import type { VaultCore } from '@/app/vault/core'

vi.mock('@/app/vault', () => ({
  core: {
    ...searchCoreMock,
    ...tabCoreMock,
  } satisfies Partial<VaultCore>,
}))
