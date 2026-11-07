/**
 * The page of secrets with nothing on it, and the control that adds a secret to a group no plugin
 * asked for. Every name and value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import {
  click,
  dispose,
  enterText,
  findButton,
  findButtons,
  findText,
  maskedInputs,
  root,
} from '@/testing/secretsDom'

const HARBOUR = 'md.numen.sample.harbour'

const plugins = [
  createPluginInfo(HARBOUR, 'Harbour', {
    secrets: [
      { name: 'apiKey', title: 'API key', description: 'Key of the harbour', group: 'tides' },
      { name: 'apiKey', title: 'API key', description: 'Key of a local harbour', group: '' },
    ],
  }),
]

const secret = (id: string, name: string, group: string) => ({
  id,
  name,
  group,
  allowed: [],
  hasValue: true,
})

const mountPage = (
  secrets: ReturnType<typeof secret>[] = [],
  overrides: Parameters<typeof createInstallation>[0] = {},
) => {
  const installation = createInstallation({
    secrets: ref(secrets),
    ...overrides,
  })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId: ref('secrets') } },
  })
  return { installation, wrapper }
}

const typeGroup = async (text: string) => {
  const input = root().querySelector<HTMLInputElement>(
    'input[aria-label="Group of the new secret"]',
  )
  if (input === null) throw new Error('no field for the group name')
  await enterText(input, text)
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

describe('the page of secrets with no group', () => {
  it('says there is no secret yet and offers to add one', () => {
    const { wrapper } = mountPage()
    expect(findText(/no secrets yet/i)).not.toBeNull()
    expect(findButton(/add secret/i)).toBeTruthy()
    dispose(wrapper)
  })

  it('does not say there is no secret when one exists', () => {
    const { wrapper } = mountPage([secret('s00000001', 'Work', 'harbour')])
    expect(findText(/no secrets yet/i)).toBeNull()
    dispose(wrapper)
  })

  it('creates a secret in a group the person types, named Default', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add secret/i))
    expect(maskedInputs()).toHaveLength(0)
    await typeGroup('Local Llama')
    const [input] = maskedInputs()
    await enterText(input as HTMLInputElement, 'made-up-llama-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      'Default',
      'made-up-llama-value',
    )
    dispose(wrapper)
  })

  it('puts the secret in the existing group when the typed name equals it', async () => {
    const { installation, wrapper } = mountPage([
      secret('s00000001', '1', 'Local Llama'),
      secret('s00000002', 'Home', 'Local Llama'),
    ])
    await click(findButtons(/add secret/i).at(-1) as HTMLElement)
    await typeGroup('Local Llama')
    const inputs = maskedInputs()
    await enterText(inputs[inputs.length - 1] as HTMLInputElement, 'made-up-second-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      '3',
      'made-up-second-value',
    )
    dispose(wrapper)
  })

  it('lists the ungrouped declarations of a plugin under a secret of any group', () => {
    const { wrapper } = mountPage([secret('s00000001', 'Work', 'Local Llama')], {
      plugins: ref(plugins),
    })
    expect(findText('Key of a local harbour')).not.toBeNull()
    dispose(wrapper)
  })
})
