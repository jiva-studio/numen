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
import type { VaultCore } from '@/app/vault/core'

vi.mock('@/app/vault', () => ({
  core: {
    ...noteCoreMock,
    ...fileCoreMock,
    ...settingsCoreMock,
    ...tabCoreMock,
  } satisfies Partial<VaultCore>,
}))
