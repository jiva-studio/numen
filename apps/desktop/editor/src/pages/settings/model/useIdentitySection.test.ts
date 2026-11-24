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
      generateWords: () => new Promise((resolve) => (finish = resolve)),
      create: vi.fn(),
      restore: vi.fn(),
      delete: vi.fn(),
      saveKit: vi.fn(),
      devices: createInertDevices(),
      keyFile: createInertKeyFile(),
    }
    let section: ReturnType<typeof useIdentitySection> | undefined
    const host = mount(
      defineComponent({
        setup() {
          section = useIdentitySection(() => installation, () => undefined)
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
