import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { asValue, type Result } from '@numen/result'
import type { IdentityInfo } from '@/entities/identity'
import type { InstallationIdentity } from '../types'
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
    let finish: (answer: Result<readonly string[], string>) => void = () => {}
    const installation: InstallationIdentity = {
      identity: ref<IdentityInfo | undefined>(undefined),
      load: vi.fn(() => Promise.resolve()),
      generateWords: () => new Promise((resolve) => (finish = resolve)),
      create: vi.fn(),
      delete: vi.fn(),
    }
    let section: ReturnType<typeof useIdentitySection> | undefined
    const host = mount(
      defineComponent({
        setup() {
          section = useIdentitySection(() => installation)
          return () => null
        },
      }),
    )
    const starting = section?.actions.startSetup()
    host.unmount()
    finish(asValue<readonly string[]>(RECOVERY_WORDS))
    await starting
    await flushPromises()
    expect(section?.setup.recoveryWords.value).toStrictEqual([])
    expect(section?.step.value).toBe('idle')
  })
})
