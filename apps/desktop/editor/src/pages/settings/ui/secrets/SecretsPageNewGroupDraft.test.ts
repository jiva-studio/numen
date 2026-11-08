/**
 * The control that adds a secret to a group the person types: the name of the first secret, a
 * failed creation and a group name that is too long. Every name and value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import { createInstallation } from '@/testing/installation'
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
  const installation = createInstallation({ secrets: ref(secrets), ...overrides })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId: ref('secrets') } },
  })
  return { installation, wrapper }
}

const groupField = (): HTMLInputElement => {
  const input = root().querySelector<HTMLInputElement>('input[aria-label="Name of the group"]')
  if (input === null) throw new Error('no field for the group name')
  return input
}

const typeGroup = async (text: string) => {
  await enterText(groupField(), text)
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

describe('the name of the first secret in a group the person types', () => {
  it('is Default', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    await click(findButton(/add secret/i))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')
    expect(installation.createSecret).toHaveBeenCalledWith(
      'Local Llama',
      'Default',
      'made-up-llama-value',
    )
    dispose(wrapper)
  })

  it('is followed by 2 in a group that has Default', async () => {
    const { installation, wrapper } = mountPage([secret('s00000001', 'Default', 'Local Llama')])
    await click(findButton(/add secret/i))
    const inputs = maskedInputs()
    await enterText(inputs[inputs.length - 1] as HTMLInputElement, 'made-up-second-value')
    expect(installation.createSecret).toHaveBeenCalledWith(
      'Local Llama',
      '2',
      'made-up-second-value',
    )
    dispose(wrapper)
  })

  it('is followed by 3 in a group that has Default and 2', async () => {
    const { installation, wrapper } = mountPage([
      secret('s00000001', 'Default', 'Local Llama'),
      secret('s00000002', '2', 'Local Llama'),
    ])
    await click(findButton(/add secret/i))
    const inputs = maskedInputs()
    await enterText(inputs[inputs.length - 1] as HTMLInputElement, 'made-up-third-value')
    expect(installation.createSecret).toHaveBeenCalledWith(
      'Local Llama',
      '3',
      'made-up-third-value',
    )
    dispose(wrapper)
  })
})

describe('a secret that fails to be created from the control that adds a secret', () => {
  it('keeps the group and the typed value in the field', async () => {
    const createSecret = vi.fn(async () => '')
    const { wrapper } = mountPage([], { createSecret })
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    await click(findButton(/add secret/i))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')

    expect(createSecret).toHaveBeenCalled()
    expect(findText('Local Llama')?.tagName).toBe('H2')
    expect(maskedInputs()).toHaveLength(1)
    expect((maskedInputs()[0] as HTMLInputElement).value).toBe('made-up-llama-value')
    dispose(wrapper)
  })
})

describe('a group name of more than 64 characters', () => {
  it('is refused on the page, the name stays', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('g'.repeat(65))
    expect(findText('New group')?.tagName).toBe('H2')
    expect(installation.createSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('is accepted at 64 characters', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('g'.repeat(64))
    await click(findButton(/add secret/i))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')

    expect(installation.createSecret).toHaveBeenCalledWith(
      'g'.repeat(64),
      'Default',
      'made-up-value',
    )
    dispose(wrapper)
  })
})

describe('a secret that fails to be created from the control that adds a secret to a group', () => {
  it('keeps the typed value in the field', async () => {
    const createSecret = vi.fn(async () => '')
    const { wrapper } = mountPage([secret('s00000001', 'Default', 'Local Llama')], { createSecret })
    await click(findButtons(/add secret/i)[0] as HTMLElement)
    const inputs = maskedInputs()
    await enterText(inputs[inputs.length - 1] as HTMLInputElement, 'made-up-section-value')

    expect(createSecret).toHaveBeenCalledWith('Local Llama', '2', 'made-up-section-value')
    const after = maskedInputs()
    expect((after[after.length - 1] as HTMLInputElement).value).toBe('made-up-section-value')
    dispose(wrapper)
  })
})

describe('the name of a group the person types', () => {
  it('loses U+0085 at its ends, as the core trims it', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('\u0085abc')
    await click(findButton(/add secret/i))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')

    expect(installation.createSecret).toHaveBeenCalledWith('abc', 'Default', 'made-up-value')
    dispose(wrapper)
  })

  it('keeps U+FEFF at its ends, as the core does', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('﻿abc')
    await click(findButton(/add secret/i))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')

    expect(installation.createSecret).toHaveBeenCalledWith('﻿abc', 'Default', 'made-up-value')
    dispose(wrapper)
  })
})
