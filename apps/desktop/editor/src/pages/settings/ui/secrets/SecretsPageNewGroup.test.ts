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
  addKey,
  chooseGroupAction,
  click,
  dispose,
  enterText,
  findButton,
  findText,
  maskedInputs,
  root,
  rowOf,
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

describe('the page of secrets with no group', () => {
  it('has no explanation and offers to add a group', () => {
    const { wrapper } = mountPage()
    expect(findText(/no secrets yet/i)).toBeNull()
    expect(findButton(/add group/i)).toBeTruthy()
    dispose(wrapper)
  })

  it('creates a group at once with a name to type over, focused and selected', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    const input = groupField()
    expect(input.value).toBe('New group')
    expect(document.activeElement).toBe(input)
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe('New group'.length)
    expect(installation.createSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('numbers the name of a new group when it is taken', async () => {
    const { wrapper } = mountPage([secret('s00000001', 'Work', 'New group')])
    await click(findButton(/add group/i))
    expect(groupField().value).toBe('New group 2')
    await click(findButton(/add group/i))
    expect(findText('New group')).not.toBeNull()
    expect(findText('New group 2')).not.toBeNull()
    dispose(wrapper)
  })

  it('keeps an empty group the person named until a secret is added, then creates the secret in it', async () => {
    const { installation, wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    expect(installation.renameSecretGroup).not.toHaveBeenCalled()
    expect(findText('Local Llama')?.tagName).toBe('H2')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      'Default',
      'made-up-llama-value',
    )
    dispose(wrapper)
  })

  it('drops an empty group the person deletes, without a question', async () => {
    const { wrapper } = mountPage()
    await click(findButton(/add group/i))
    await typeGroup('New group')
    await chooseGroupAction(/delete group/i)
    expect(findText('New group')).toBeNull()
    expect(root().querySelector('[role="dialog"], [role="alertdialog"], dialog')).toBeNull()
    dispose(wrapper)
  })

  it('puts the next secret in the existing group', async () => {
    const { installation, wrapper } = mountPage([
      secret('s00000001', '1', 'Local Llama'),
      secret('s00000002', 'Home', 'Local Llama'),
    ])
    await addKey()
    const inputs = maskedInputs()
    await enterText(inputs[inputs.length - 1] as HTMLInputElement, 'made-up-second-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      'Default V1',
      'made-up-second-value',
    )
    dispose(wrapper)
  })

  it('lists only the ungrouped declarations of a plugin under a secret of a group no plugin asks for', async () => {
    const { wrapper } = mountPage([secret('s00000001', 'Work', 'Local Llama')], {
      plugins: ref(plugins),
    })
    await click(findButton(/plugin/i, rowOf('Work')))
    const names = Array.from(root().querySelectorAll('[role="option"]')).map((one) =>
      one.textContent?.trim(),
    )
    expect(names).toStrictEqual(['Harbour'])
    dispose(wrapper)
  })
})
