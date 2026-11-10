/**
 * The secret fields on the page of a plugin: a select of the secrets for each declaration, with a
 * search over the secrets, and a button to the page of secrets where the declared group
 * has none. The plugin, the secrets and every value here are invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { getServiceInfo } from '@numen/core-client'
import { mount } from '@vue/test-utils'
import PluginDetail from './PluginDetail.vue'
import SettingsTab from '../SettingsTab.vue'
import {
  createInstallation,
  createPluginInfo,
  createSecretAnswer,
  createSettingsState,
} from '@/pages/settings/testing/installation'
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
  const installation = createInstallation({
    plugins: ref([plugin]),
    ...createSecretAnswer([plugin], secrets),
  })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: createSettingsState(installation, { initialPluginId }) },
  })
  return { installation, initialPluginId, wrapper }
}

/** The CSS selector of the select button in a row. */
const SELECT = 'button[data-slot="select"]'

const fieldOf = (title: string): HTMLElement => {
  let one = (findText(title) as HTMLElement).parentElement
  while (one !== null && one.querySelector(SELECT) === null) one = one.parentElement
  if (one === null) throw new Error(`no field for ${title}`)
  return one
}

const openMenu = async (title: string): Promise<void> => {
  await click(fieldOf(title).querySelector(SELECT) as HTMLElement)
}

const optionNames = (): string[] =>
  Array.from(root().querySelectorAll('[role="option"] .searchable-list__text')).map((one) =>
    (one.textContent ?? '').trim(),
  )

const menuItem = (name: string): HTMLElement => {
  const found = Array.from(root().querySelectorAll<HTMLElement>('[role="option"]')).filter(
    (one) => one.querySelector('.searchable-list__text')?.textContent?.trim() === name,
  )
  if (found.length !== 1) throw new Error(`${found.length} rows named ${name}`)
  return found[0] as HTMLElement
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
  it('is a select that opens the list of secrets', async () => {
    const { wrapper } = mountDetail()
    expect(fieldOf('API key').querySelectorAll(SELECT)).toHaveLength(1)
    expect(optionNames()).toStrictEqual([])
    await openMenu('API key')
    expect(optionNames()).not.toHaveLength(0)
    dispose(wrapper)
  })

  it('shows the secrets allowed and the stand-in where there is none', () => {
    const { wrapper } = mountDetail()
    expect(fieldOf('API key').querySelector(SELECT)?.textContent?.trim()).toBe('Chat')
    expect(fieldOf('Organization').querySelector(SELECT)?.textContent?.trim()).toBe(
      'No secret allowed',
    )
    dispose(wrapper)
  })

  it('has no field to type a value into, and shows no secret value', async () => {
    const { installation, wrapper } = mountDetail()
    expect(fieldOf('API key').querySelectorAll('input')).toHaveLength(0)
    expect(maskedInputs()).toHaveLength(0)
    await openMenu('API key')
    expect(installation.secrets.edit.createSecret).not.toHaveBeenCalled()
    expect(installation.secrets.edit.writeSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('puts the group of each secret under its name', async () => {
    const { wrapper } = mountDetail()
    await openMenu('API key')
    const details = Array.from(root().querySelectorAll('[role="option"]')).map((one) =>
      one.querySelector('.searchable-list__detail')?.textContent?.trim(),
    )
    expect(details).toStrictEqual([
      getServiceInfo('openai').title,
      getServiceInfo('openai').title,
      getServiceInfo('openrouter').title,
      getServiceInfo('openrouter').title,
      getServiceInfo('travel').title,
    ])
    dispose(wrapper)
  })

  it('puts the secrets of the declared group on top and the others below in the order of the page', async () => {
    const { wrapper } = mountDetail()
    await openMenu('API key')
    expect(optionNames()).toStrictEqual(['Chat', 'Spare', 'Work', 'Home', 'Personal'])
    dispose(wrapper)
  })

  it('shows all the secrets in the order of the page for a declaration with no group', async () => {
    const { wrapper } = mountDetail()
    await openMenu('Organization')
    expect(optionNames()).toStrictEqual(['Chat', 'Spare', 'Work', 'Home', 'Personal'])
    dispose(wrapper)
  })

  it('narrows the secrets below the declared group by a search on the name', async () => {
    const { wrapper } = mountDetail()
    await openMenu('API key')
    const search = root().querySelector<HTMLInputElement>(
      'input[type="search"], input[placeholder*="earch" i]',
    ) as HTMLInputElement
    await enterText(search, 'hom')
    expect(optionNames()).toStrictEqual(['Home'])
    dispose(wrapper)
  })

  it('allows the declaration to read a secret by a click, as the check box does', async () => {
    const { installation, wrapper } = mountDetail()
    await openMenu('API key')
    await click(menuItem('Spare'))
    expect(installation.secrets.access.allowPluginSecret).toHaveBeenCalledExactlyOnceWith(
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
    expect(installation.secrets.access.denyPluginSecret).toHaveBeenCalledExactlyOnceWith(
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
    expect(installation.secrets.access.allowPluginSecret).toHaveBeenCalledExactlyOnceWith(
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
    const installation = createInstallation({
      plugins: ref([plugin]),
      ...createSecretAnswer([plugin], [WORK]),
    })
    const wrapper = mount(PluginDetail, {
      attachTo: document.body,
      props: {
        settings: installation.settings,
        manager: installation.plugin,
        choices: installation.secrets,
        plugin,
      },
    })
    expect(root().querySelectorAll(SELECT)).toHaveLength(3)
    dispose(wrapper)
  })
})
