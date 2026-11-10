/**
 * The vault mock the window uses, responding from `vaultState` and recording requests
 * in `requests`. Each entity's answers are in `mocks/<entity>.ts`.
 */
import { vi } from 'vitest'
import './mocks/plugin'
import { fileCoreMock } from './mocks/file'
import { noteCoreMock } from './mocks/note'
import { settingsCoreMock } from './mocks/settings'
import { tabCoreMock } from './mocks/tab'
import { vaultCoreMock, vaultsMock } from './mocks/vault'
import { windowCoreMock } from './mocks/window'
import type { VaultCore } from '@/app/vault/core'

vi.mock('@/app/vault', () => ({
  vaults: vaultsMock,
  core: {
    ...vaultCoreMock,
    ...noteCoreMock,
    ...fileCoreMock,
    ...settingsCoreMock,
    ...tabCoreMock,
    ...windowCoreMock,
  } satisfies Partial<VaultCore>,
}))
