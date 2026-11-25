import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { asValue, type Result } from '@numen/result'
import type { IdentityInfo } from '@/entities/identity'
import type { InstallationIdentity } from '../types'
import { createInertDevices, createInertKeyFile } from '../testing/installation'
import { useIdentitySection } from './useIdentitySection'

const RECOVERY_WORDS = [
  'legal',
  'winner',
  'thank',
  'year',
  'wave',
  'sausage',
  'worth',
  'useful',
  'exact',
  'glove',
  'honey',
  'yellow',
]

describe('the identity section', () => {
  it('drops the words generated after the section left the page', async () => {
    let finish = undefined as ((answer: Result<readonly string[], string>) => void) | undefined
    const installation: InstallationIdentity = {
      identity: ref<IdentityInfo | undefined>(undefined),
      load: vi.fn(() => Promise.resolve()),
      create: vi.fn(),
      delete: vi.fn(),
      recovery: {
        generateWords: () => new Promise((resolve) => (finish = resolve)),
        restore: vi.fn(),
        saveKit: vi.fn(),
      },
      devices: createInertDevices(),
      keyFile: createInertKeyFile(),
    }
    let section: ReturnType<typeof useIdentitySection> | undefined
    const host = mount(
      defineComponent({
        setup() {
          section = useIdentitySection(
            () => installation,
            () => undefined,
          )
          return () => null
        },
      }),
    )
    const starting = section?.actions.startSetup()
    host.unmount()
    finish?.(asValue<readonly string[]>(RECOVERY_WORDS))
    await starting
    await flushPromises()
    expect(section?.setup.recoveryWords.value).toStrictEqual([])
    expect(section?.step.value).toBe('idle')
  })
})

describe('the identity section restoring from words', () => {
  it('aborts the wait for the restored identity when the person cancels', async () => {
    let signal: AbortSignal | undefined
    const installation: InstallationIdentity = {
      identity: ref<IdentityInfo | undefined>(undefined),
      load: vi.fn(() => Promise.resolve()),
      create: vi.fn(),
      delete: vi.fn(),
      recovery: {
        generateWords: vi.fn(),
        restore: (_proof, given) => {
          signal = given
          return new Promise(() => undefined)
        },
        saveKit: vi.fn(),
      },
      devices: createInertDevices(),
      keyFile: createInertKeyFile(),
    }
    let section: ReturnType<typeof useIdentitySection> | undefined
    mount(
      defineComponent({
        setup() {
          section = useIdentitySection(
            () => installation,
            () => undefined,
          )
          return () => null
        },
      }),
    )
    section?.actions.startRestore('restore-words')
    section?.restore.words.setText(RECOVERY_WORDS.slice(0, 12).join(' '))
    section?.actions.restore().catch(() => undefined)
    await flushPromises()
    expect(signal?.aborted).toBe(false)
    section?.actions.cancel()
    expect(signal?.aborted).toBe(true)
  })

  it('shows no problem when the person cancels the restore that was waiting', async () => {
    const installation: InstallationIdentity = {
      identity: ref<IdentityInfo | undefined>(undefined),
      load: vi.fn(() => Promise.resolve()),
      create: vi.fn(),
      delete: vi.fn(),
      recovery: {
        generateWords: vi.fn(),
        restore: (_proof, given) =>
          new Promise((_resolve, reject) => {
            given?.addEventListener('abort', () => reject(given.reason), { once: true })
          }),
        saveKit: vi.fn(),
      },
      devices: createInertDevices(),
      keyFile: createInertKeyFile(),
    }
    let section: ReturnType<typeof useIdentitySection> | undefined
    mount(
      defineComponent({
        setup() {
          section = useIdentitySection(
            () => installation,
            () => undefined,
          )
          return () => null
        },
      }),
    )
    section?.actions.startRestore('restore-words')
    section?.restore.words.setText(RECOVERY_WORDS.slice(0, 12).join(' '))
    section?.actions.restore().catch(() => undefined)
    await flushPromises()
    section?.actions.cancel()
    await flushPromises()
    expect(section?.call.problem.value).toBe('')
  })
})
