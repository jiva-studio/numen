/**
 * The secret fields on the page of a plugin: a masked field with an arrow for each declaration, a
 * menu of the secrets with a search, and a button to the page of secrets where the declared group
 * has none. The plugin, the secrets and every value here are invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import PluginDetail from './PluginDetail.vue'
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
  positionOf,
  root,
} from '@/testing/secretsDom'

const PLUGIN = 'md.numen.sample.embeddings'

const plugin = createPluginInfo(PLUGIN, 'Embeddings', {
  secrets: [
    { name: 'apiKey', title: 'API key', description: 'Key of the service', group: 'openai' },
    { name: 'orgId', title: 'Organization', description: 'Organization of the account', group: '' },
    {
      name: 'token',
      title: 'Token',
      description: 'Token of the mistral service',
      group: 'mistral',
    },
  ],
})

const secret = (
  id: string,
  name: string,
  group: string,
  allowed: { pluginId: string; name: string; group: string }[] = [],
) => ({ id, name, group, allowed, hasValue: true })

const WORK = secret('s00000001', 'Work', 'openrouter')
const HOME = secret('s00000002', 'Home', 'openrouter')
const CHAT = secret('s00000003', 'Chat', 'openai', [
  { pluginId: PLUGIN, name: 'apiKey', group: 'openai' },
])
const SPARE = secret('s00000004', 'Spare', 'openai')
const MINE = secret('s00000005', 'Personal', 'travel')

const mountDetail = (secrets = [WORK, HOME, CHAT, SPARE, MINE]) => {
  const initialPluginId = ref<string | null>(PLUGIN)
  const installation = createInstallation({ plugins: ref([plugin]), secrets: ref(secrets) })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId } },
  })
  return { installation, initialPluginId, wrapper }
}

/** The field of a declaration: the smallest element around its title that holds a masked input. */
const fieldOf = (title: string): HTMLElement => {
  let one = (findText(title) as HTMLElement).parentElement
  while (one !== null && maskedInputs(one).length === 0) one = one.parentElement
  if (one === null) throw new Error(`no field for ${title}`)
  return one
}

const openMenu = async (title: string): Promise<void> => {
  await click(findButton(/choose/i, fieldOf(title)))
}

const menuItem = (name: string): HTMLElement => {
  const text = findText(name) as HTMLElement
  return (text.closest(
    '[role="menuitemcheckbox"], [role="menuitem"], [role="option"], [role="checkbox"], label, button, li',
  ) ?? text) as HTMLElement
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

describe('the secret field of a declaration', () => {
  it('is a masked field with an arrow that opens a menu', async () => {
    const { wrapper } = mountDetail()
    expect(maskedInputs(fieldOf('API key'))).toHaveLength(1)
    expect(findButtons(/choose/i, fieldOf('API key'))).toHaveLength(1)
    expect(findText('Work')).toBeNull()
    await openMenu('API key')
    expect(findText('Work')).not.toBeNull()
    dispose(wrapper)
  })

  it('takes no typed value', async () => {
    const { installation, wrapper } = mountDetail()
    const input = maskedInputs(fieldOf('API key'))[0] as HTMLInputElement
    await enterText(input, 'made-up-typed-value')
    expect(installation.createSecret).not.toHaveBeenCalled()
    expect(installation.writeSecret).not.toHaveBeenCalled()
    expect(root().innerHTML).not.toContain('made-up-typed-value')
    dispose(wrapper)
  })

  it('shows no secret value', () => {
    const { wrapper } = mountDetail()
    for (const input of maskedInputs()) {
      expect(input.value).toBe('')
    }
    dispose(wrapper)
  })

  it('puts the secrets of the declared group on top and the others below in the order of the page', async () => {
    const { wrapper } = mountDetail()
    await openMenu('API key')
    const order = ['Chat', 'Spare', 'Work', 'Home', 'Personal'].map((name) =>
      positionOf(findText(name) as HTMLElement),
    )
    expect(order).toStrictEqual([...order].sort((a, b) => a - b))
    dispose(wrapper)
  })

  it('shows all the secrets in the order of the page for a declaration with no group', async () => {
    const { wrapper } = mountDetail()
    await openMenu('Organization')
    const order = ['Chat', 'Spare', 'Work', 'Home', 'Personal'].map((name) =>
      positionOf(findText(name) as HTMLElement),
    )
    expect(order).toStrictEqual([...order].sort((a, b) => a - b))
    dispose(wrapper)
  })

  it('narrows the secrets below the declared group by a search on the name', async () => {
    const { wrapper } = mountDetail()
    await openMenu('API key')
    const search = root().querySelector<HTMLInputElement>(
      'input[type="search"], input[placeholder*="earch" i]',
    ) as HTMLInputElement
    await enterText(search, 'hom')
    expect(findText('Home')).not.toBeNull()
    expect(findText('Work')).toBeNull()
    expect(findText('Personal')).toBeNull()
    dispose(wrapper)
  })

  it('allows the declaration to read a secret by a click, as the check box does', async () => {
    const { installation, wrapper } = mountDetail()
    await openMenu('API key')
    await click(menuItem('Spare'))
    expect(installation.allowPluginSecret).toHaveBeenCalledExactlyOnceWith(
      SPARE.id,
      PLUGIN,
      'apiKey',
      'openai',
    )
    dispose(wrapper)
  })

  it('denies the declaration a secret it may read by a click', async () => {
    const { installation, wrapper } = mountDetail()
    await openMenu('API key')
    await click(menuItem('Chat'))
    expect(installation.denyPluginSecret).toHaveBeenCalledExactlyOnceWith(
      CHAT.id,
      PLUGIN,
      'apiKey',
      'openai',
    )
    dispose(wrapper)
  })

  it('passes an empty group for a declaration with no group', async () => {
    const { installation, wrapper } = mountDetail()
    await openMenu('Organization')
    await click(menuItem('Work'))
    expect(installation.allowPluginSecret).toHaveBeenCalledExactlyOnceWith(
      WORK.id,
      PLUGIN,
      'orgId',
      '',
    )
    dispose(wrapper)
  })

  it('shows a button that opens the page of secrets where the declared group has no secret', async () => {
    const { initialPluginId, wrapper } = mountDetail()
    const field = fieldOf('Token')
    await click(findButton(/secrets/i, field))
    expect(initialPluginId.value).toBe('secrets')
    dispose(wrapper)
  })

  it('shows no such button where the declared group has a secret', () => {
    const { wrapper } = mountDetail()
    expect(findButtons(/open.*secrets|secrets page/i, fieldOf('API key'))).toHaveLength(0)
    dispose(wrapper)
  })
})

describe('the page of a plugin', () => {
  it('still mounts on its own with the secret fields of its declarations', () => {
    const installation = createInstallation({ plugins: ref([plugin]), secrets: ref([WORK]) })
    const wrapper = mount(PluginDetail, {
      attachTo: document.body,
      props: { state: { installation }, plugin },
    })
    expect(maskedInputs()).toHaveLength(3)
    dispose(wrapper)
  })
})
