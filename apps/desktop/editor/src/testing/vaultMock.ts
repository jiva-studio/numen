/**
 * The vault mock the window uses, responding from `vaultState` and recording requests
 * in `requests`. Each entity's answers are in `mocks/<entity>.ts`: an entity that calls the core
 * through its own `api` is answered through the router transport, and the rest through the mocked
 * `core` object.
 */
import { afterEach, beforeEach, vi } from 'vitest'
import { installCoreAnswers, type CoreAnswers } from '@numen/core-client/testing'
import './mocks/plugin'
import { fileAnswers } from './mocks/file'
import { noteAnswers } from './mocks/note'
import { searchCoreMock } from './mocks/search'
import { settingsCoreMock } from './mocks/settings'
import { tabCoreMock } from './mocks/tab'
import { vaultCoreMock, vaultsMock } from './mocks/vault'
import { windowCoreMock } from './mocks/window'
import type { VaultCore } from '@/app/vault/core'

const answers: CoreAnswers = {
  notes: noteAnswers,
  files: fileAnswers,
}

let restoreAnswers: () => void

beforeEach(() => {
  restoreAnswers = installCoreAnswers(answers)
})

afterEach(() => {
  restoreAnswers()
})

vi.mock('@/app/vault', () => ({
  vaults: vaultsMock,
  core: {
    ...vaultCoreMock,
    ...searchCoreMock,
    ...settingsCoreMock,
    ...tabCoreMock,
    ...windowCoreMock,
  } satisfies Partial<VaultCore>,
}))
