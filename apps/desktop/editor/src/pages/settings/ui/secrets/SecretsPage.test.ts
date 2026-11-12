/**
 * The page of secrets in the settings tab, opened by the id `secrets` as the shared settings are
 * opened by `system`. The secrets, the plugins and every value here are invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, ref, type Ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import {
  createInstallation,
  createPluginInfo,
  createSecretAnswer,
  createSettingsState,
} from '@/pages/settings/testing/installation'
import {
  addSecret,
  chooseGroupAction,
  chooseRowAction,
  click,
  dispose,
  enterText,
  findButton,
  findButtons,
  findMenuItem,
  findNameInput,
  findText,
  maskedInputs,
  nameOf,
  openRowMenu,
  positionOf,
  root,
  rowOf,
  sectionOf,
  textInputs,
  typeName,
} from '@/testing/secretsDom'

const EMBEDDINGS = 'md.numen.sample.embeddings'
const PROOFREADING = 'md.numen.sample.proofreading'
const HARBOUR = 'md.numen.sample.harbour'

const plugins = [
  createPluginInfo(EMBEDDINGS, 'Embeddings', {
    secrets: [
      {
        name: 'apiKey',
        title: 'API key',
        description: 'Key of the embeddings service',
        group: 'openrouter',
      },
      {
        name: 'apiKey',
        title: 'API key',
        description: 'Key of the custom embeddings service',
        group: '',
      },
    ],
  }),
  createPluginInfo(PROOFREADING, 'Proofreading', {
    secrets: [
      {
        name: 'apiKey',
        title: 'API key',
        description: 'Key of the proofreading service',
        group: 'openrouter',
      },
      {
        name: 'apiKey',
        title: 'API key',
        description: 'Key of the proofreading chat',
        group: 'openai',
      },
    ],
  }),
]

const harbour = createPluginInfo(HARBOUR, 'Harbour', {
  secrets: [
    { name: 'apiKey', title: 'API key', description: 'Key of the harbour', group: 'tides' },
    { name: 'apiKey', title: 'API key', description: 'Key of a local harbour', group: '' },
  ],
})

const secret = (
  id: string,
  name: string,
  group: string,
  allowed: { pluginId: string; name: string; group: string }[] = [],
  hasValue = true,
) => ({ id, name, group, allowed, hasValue })

const WORK = secret('s00000001', 'Work', 'openrouter', [
  { pluginId: EMBEDDINGS, name: 'apiKey', group: 'openrouter' },
])
const HOME = secret('s00000002', 'Home', 'openrouter')
const MINE = secret('s00000003', 'Personal', 'travel')

const mountPage = (
  secrets = [WORK, HOME, MINE],
  overrides: Parameters<typeof createInstallation>[0] = {},
) => {
  const pluginList = overrides.plugins?.value ?? plugins
  const installation = createInstallation({
    plugins: ref(pluginList),
    ...createSecretAnswer(pluginList, secrets),
    ...overrides,
  })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: createSettingsState(installation, { initialPluginId: ref('secrets') }) },
  })
  return { installation, wrapper }
}

const liveAnswer = (stored: Ref<ReturnType<typeof secret>[]>) => ({
  sections: computed(() => createSecretAnswer([], stored.value).sections?.value ?? []),
  menus: ref([]),
  states: computed(() => createSecretAnswer([], stored.value).states?.value ?? new Map()),
})

const mountBare = (
  secrets: ReturnType<typeof secret>[] = [],
  overrides: Parameters<typeof createInstallation>[0] = {},
) => mountPage(secrets, { plugins: ref([]), ...overrides })

const groupField = (): HTMLInputElement => {
  const input = root().querySelector<HTMLInputElement>('input[aria-label="Name of the group"]')
  if (input === null) throw new Error('no field for the group name')
  return input
}

const typeGroup = (text: string) => enterText(groupField(), text)

const lastMasked = (): HTMLInputElement => maskedInputs().at(-1) as HTMLInputElement

const getDescription = (input: HTMLElement): string =>
  (input.getAttribute('aria-describedby') ?? '')
    .split(' ')
    .map((id) => document.getElementById(id)?.textContent?.trim() ?? '')
    .join(' ')
    .trim()

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

describe('the sections of the page', () => {
  it('has a section for each group, the groups plugins ask for by name and then the ones the person made by name', () => {
    const { wrapper } = mountPage()
    const order = ['OpenAI', 'OpenRouter', 'travel'].map((group) => sectionOf(group))
    const positions = order.map((one) => positionOf(one))
    expect(positions).toStrictEqual([...positions].sort((a, b) => a - b))
    dispose(wrapper)
  })

  it('lists the secrets of a group in priority order', () => {
    const { wrapper } = mountPage()
    expect(positionOf(rowOf('Work'))).toBeLessThan(positionOf(rowOf('Home')))
    dispose(wrapper)
  })

  it('shows a group a plugin asks for with no secret as one empty secret named after its service', () => {
    const { wrapper } = mountPage()
    expect(sectionOf('OpenAI').contains(rowOf('OpenAI'))).toBe(true)
    dispose(wrapper)
  })

  it('stores the empty secret under its group on the first entry of a value', async () => {
    const { installation, wrapper } = mountPage()
    await enterText(maskedInputs(rowOf('OpenAI'))[0] as HTMLInputElement, 'made-up-first-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openai',
      'OpenAI',
      'made-up-first-value',
    )
    dispose(wrapper)
  })

  it('shows the path of the file where the values are kept in a file', () => {
    const path = '/home/someone/.config/numen/credentials/credentials.json'
    const { wrapper } = mountPage([WORK], { credentialsPath: ref(path) })
    expect(findText(/Values are kept in/)?.textContent).toContain(path)
    dispose(wrapper)
  })

  it('says nothing about the place where no file is used', () => {
    const { wrapper } = mountPage([WORK], { credentialsPath: ref('') })
    expect(findText(/Values are kept in/)).toBeNull()
    dispose(wrapper)
  })
})

describe('the rows of a group', () => {
  it('moves a secret up before the one above it and down after the one below it', async () => {
    const { installation, wrapper } = mountPage()
    await chooseRowAction(rowOf('Home'), /move up/i)
    expect(installation.secrets.edit.moveSecret).toHaveBeenLastCalledWith(HOME.id, WORK.id)

    await chooseRowAction(rowOf('Work'), /move down/i)
    expect(installation.secrets.edit.moveSecret).toHaveBeenLastCalledWith(WORK.id, '')
    dispose(wrapper)
  })

  it('cannot move the first secret of a group up or the last one down', async () => {
    const first = mountPage()
    const itemsOfWork = await openRowMenu(rowOf('Work'))
    expect(itemsOfWork.find((one) => /move up/i.test(nameOf(one)))?.hasAttribute('disabled')).toBe(
      true,
    )
    dispose(first.wrapper)

    const second = mountPage()
    const itemsOfHome = await openRowMenu(rowOf('Home'))
    expect(
      itemsOfHome.find((one) => /move down/i.test(nameOf(one)))?.hasAttribute('disabled'),
    ).toBe(true)
    dispose(second.wrapper)
  })

  it('offers the order items only in a group with more than one secret', async () => {
    const first = mountPage()
    expect((await openRowMenu(rowOf('Work'))).map(nameOf)).toEqual([
      'Move up',
      'Move down',
      'Delete',
    ])
    dispose(first.wrapper)

    const second = mountPage()
    expect((await openRowMenu(rowOf('Personal'))).map(nameOf)).toEqual(['Delete'])
    dispose(second.wrapper)
  })

  it('renames a secret by its id', async () => {
    const { installation, wrapper } = mountPage()
    expect(findButtons(/rename/i, rowOf('Work'))).toHaveLength(0)
    await typeName(findNameInput('Work') as HTMLInputElement, 'Office')
    expect(installation.secrets.edit.renameSecret).toHaveBeenCalledExactlyOnceWith(
      WORK.id,
      'Office',
    )
    dispose(wrapper)
  })

  it('trims the name of a secret as the core does', async () => {
    const { installation, wrapper } = mountPage()
    await typeName(findNameInput('Work') as HTMLInputElement, '\u0085Office\u0085')
    expect(installation.secrets.edit.renameSecret).toHaveBeenCalledExactlyOnceWith(
      WORK.id,
      'Office',
    )
    dispose(wrapper)
  })
})

describe('the value of a secret', () => {
  it('shows dots as the placeholder and says the value is set while one is stored', () => {
    const { wrapper } = mountPage([WORK])
    const input = maskedInputs(rowOf('Work'))[0] as HTMLInputElement
    expect(input.placeholder).toBe('••••••••')
    expect(input.value).toBe('')
    expect(getDescription(input)).toBe('Value is set')
    dispose(wrapper)
  })

  it('shows a placeholder and says there is no value yet where none is stored', () => {
    const { wrapper } = mountPage([secret('s00000009', 'Imported', 'openrouter', [], false)])
    const input = maskedInputs(rowOf('Imported'))[0] as HTMLInputElement
    expect(input.placeholder).toBe('Paste a value')
    expect(getDescription(input)).toBe('No value yet')
    dispose(wrapper)
  })

  it('writes a value entered over a stored one', async () => {
    const { installation, wrapper } = mountPage([WORK])
    await enterText(maskedInputs(rowOf('Work'))[0] as HTMLInputElement, 'made-up-new-value')
    expect(installation.secrets.edit.writeSecret).toHaveBeenCalledExactlyOnceWith(
      WORK.id,
      'made-up-new-value',
    )
    dispose(wrapper)
  })

  it('shows no value after it is entered', async () => {
    const { installation, wrapper } = mountPage()
    const input = maskedInputs(rowOf('OpenAI'))[0] as HTMLInputElement
    await enterText(input, 'made-up-value-that-must-not-stay')

    expect(installation.secrets.edit.createSecret).toHaveBeenCalledOnce()
    expect(input.value).toBe('')
    expect(root().innerHTML).not.toContain('made-up-value-that-must-not-stay')
    dispose(wrapper)
  })
})

describe('the plugins that may read a secret', () => {
  const getReaders = (row: HTMLElement) =>
    row.querySelector<HTMLElement>('button[data-slot="select"]') as HTMLElement
  const optionOf = (pluginName: string) =>
    Array.from(root().querySelectorAll<HTMLElement>('[role="option"]')).find(
      (one) => one.textContent?.trim() === pluginName,
    ) as HTMLElement
  const isMarked = (option: HTMLElement) => option.getAttribute('aria-current') === 'true'
  const optionNames = () =>
    Array.from(root().querySelectorAll('[role="option"]')).map((one) => one.textContent?.trim())

  it('lists a plugin once per name: its declarations of the group, and the ungrouped one only where it has none for the group', async () => {
    const { wrapper } = mountPage()
    await click(getReaders(rowOf('Work')))
    expect(optionNames()).toStrictEqual(['Embeddings', 'Proofreading'])
    dispose(wrapper)
  })

  it('lists the ungrouped declarations under a secret of a group no plugin asks for', async () => {
    const { wrapper } = mountPage()
    await click(getReaders(rowOf('Personal')))
    expect(optionNames()).toStrictEqual(['Embeddings'])
    dispose(wrapper)
  })

  it('names the plugin that may read the secret on the button, and the stand-in where none may', () => {
    const { wrapper } = mountPage()
    expect(getReaders(rowOf('Work')).textContent).toContain('Embeddings')
    expect(getReaders(rowOf('Home')).textContent).toContain('No plugins')
    dispose(wrapper)
  })

  it('marks an allowed plugin with a check and toggles it with allow and deny', async () => {
    const { installation, wrapper } = mountPage()
    await click(getReaders(rowOf('Work')))
    expect(isMarked(optionOf('Embeddings'))).toBe(true)
    expect(isMarked(optionOf('Proofreading'))).toBe(false)

    await click(optionOf('Proofreading'))
    expect(installation.secrets.access.allowPluginSecret).toHaveBeenLastCalledWith(
      WORK.id,
      PROOFREADING,
      'apiKey',
      'openrouter',
    )
    await click(optionOf('Embeddings'))
    expect(installation.secrets.access.denyPluginSecret).toHaveBeenLastCalledWith(
      WORK.id,
      EMBEDDINGS,
      'apiKey',
      'openrouter',
    )
    dispose(wrapper)
  })

  it('lists only the ungrouped declarations of a plugin under a secret of a group no plugin asks for', async () => {
    const { wrapper } = mountBare([secret('s00000001', 'Work', 'Local Llama')], {
      plugins: ref([harbour]),
    })
    await click(getReaders(rowOf('Work')))
    expect(optionNames()).toStrictEqual(['Harbour'])
    dispose(wrapper)
  })
})

describe('deleting a secret', () => {
  const getDialog = () => root().querySelector('[role="alertdialog"]') as HTMLElement

  it('asks for a confirmation that names the plugins that can read the secret before it deletes one', async () => {
    const { installation, wrapper } = mountPage()
    await chooseRowAction(rowOf('Work'), /^delete$/i)

    expect(installation.secrets.edit.deleteSecret).not.toHaveBeenCalled()
    expect(getDialog().textContent).toContain('Embeddings')
    expect(getDialog().textContent).not.toContain('Proofreading')

    await click(findButton(/^delete$/i, getDialog()))
    expect(installation.secrets.edit.deleteSecret).toHaveBeenCalledExactlyOnceWith(WORK.id)
    dispose(wrapper)
  })

  it('describes the dialog by its question', async () => {
    const { wrapper } = mountPage()
    await chooseRowAction(rowOf('Work'), /^delete$/i)
    const describedBy = getDialog().getAttribute('aria-describedby') ?? ''
    expect(document.getElementById(describedBy)?.textContent).toContain('Delete Work?')
    dispose(wrapper)
  })

  it('reads the states of the values when the page opens', () => {
    const { installation, wrapper } = mountPage()
    expect(installation.secrets.loadStates).toHaveBeenCalledOnce()
    dispose(wrapper)
  })

  it('deletes a whole group with one call after the confirmation', async () => {
    const { installation, wrapper } = mountPage()
    await chooseGroupAction(/delete group/i, sectionOf('travel'))
    await click(findButton(/^delete$/i, getDialog()))
    expect(installation.secrets.edit.deleteSecretGroup).toHaveBeenCalledExactlyOnceWith('travel')
    expect(installation.secrets.edit.deleteSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('deletes nothing when the confirmation is cancelled', async () => {
    const { installation, wrapper } = mountPage()
    await chooseRowAction(rowOf('Work'), /^delete$/i)
    await click(findButton(/cancel/i, getDialog()))
    expect(installation.secrets.edit.deleteSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('returns the focus to the button that opened the confirmation after Cancel', async () => {
    const { wrapper } = mountPage()
    const opener = findButton(/secret actions/i, rowOf('Home'))
    opener.focus()
    await click(opener)
    await click(findMenuItem(/^delete$/i))
    await click(findButton(/cancel/i, getDialog()))
    expect(document.activeElement).toBe(opener)
    dispose(wrapper)
  })

  it('returns the focus to the button that opened the confirmation after Confirm while the button is still there', async () => {
    const { wrapper } = mountPage()
    const opener = findButton(/secret actions/i, rowOf('Home'))
    opener.focus()
    await click(opener)
    await click(findMenuItem(/^delete$/i))
    await click(findButton(/^delete$/i, getDialog()))
    expect(document.activeElement).toBe(opener)
    dispose(wrapper)
  })
})

describe('adding a secret to a group', () => {
  it('names it with the next version and stores it when a value is entered', async () => {
    const { installation, wrapper } = mountPage()
    await addSecret(sectionOf('OpenRouter'))
    expect(findNameInput('OpenRouter V1')).not.toBeNull()
    await enterText(maskedInputs(rowOf('OpenRouter V1'))[0] as HTMLInputElement, 'made-up-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openrouter',
      'OpenRouter V1',
      'made-up-value',
    )
    dispose(wrapper)
  })

  it('stores it under the name typed into its name input', async () => {
    const { installation, wrapper } = mountPage()
    await addSecret(sectionOf('OpenRouter'))
    await typeName(findNameInput('OpenRouter V1') as HTMLInputElement, 'Staging')
    await enterText(maskedInputs(rowOf('Staging'))[0] as HTMLInputElement, 'made-up-staging-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openrouter',
      'Staging',
      'made-up-staging-value',
    )
    dispose(wrapper)
  })

  it('names the second secret of a group no service is known by Default V1', async () => {
    const { wrapper } = mountPage()
    await addSecret(sectionOf('travel'))
    expect(findNameInput('Default V1')).not.toBeNull()
    dispose(wrapper)
  })

  it('keeps its plugin list disabled until the secret exists', async () => {
    const { wrapper } = mountPage()
    await addSecret(sectionOf('OpenRouter'))
    const draft = rowOf('OpenRouter V1')
    expect(findButton(/save a value first/i, draft).hasAttribute('disabled')).toBe(true)
    expect(draft.querySelector('button[data-slot="select"]')?.hasAttribute('disabled')).toBe(true)
    dispose(wrapper)
  })

  it('moves the focus into its name', async () => {
    const { wrapper } = mountPage()
    await addSecret(sectionOf('OpenRouter'))
    expect(document.activeElement).toBe(findNameInput('OpenRouter V1'))
    dispose(wrapper)
  })

  it('discards it with its button and returns the focus to the Add secret button', async () => {
    const { wrapper } = mountPage()
    const openrouter = sectionOf('OpenRouter')
    await addSecret(openrouter)
    await click(findButton(/discard/i, rowOf('OpenRouter V1')))
    expect(findNameInput('OpenRouter V1')).toBeNull()
    expect(document.activeElement).toBe(findButton(/^add secret$/i, openrouter))
    dispose(wrapper)
  })

  it('leaves exactly one row for a secret created under a name other than the one offered', async () => {
    const stored = ref([WORK, HOME, MINE])
    const createSecret = vi.fn(async (group: string, name: string) => {
      stored.value = [...stored.value, secret('s00000009', name, group)]
      return 's00000009'
    })
    const { wrapper } = mountPage(undefined, { ...liveAnswer(stored), createSecret })
    await addSecret(sectionOf('OpenRouter'))
    await typeName(findNameInput('OpenRouter V1') as HTMLInputElement, 'Staging')
    await enterText(maskedInputs(rowOf('Staging'))[0] as HTMLInputElement, 'made-up-staging-value')
    expect(textInputs(sectionOf('OpenRouter')).map((one) => one.value)).toStrictEqual([
      'Work',
      'Home',
      'Staging',
    ])
    dispose(wrapper)
  })

  it('discards one of two new secrets that share a typed name', async () => {
    const { wrapper } = mountPage()
    const openrouter = sectionOf('OpenRouter')
    await addSecret(openrouter)
    await addSecret(openrouter)
    for (const input of textInputs(openrouter).filter((one) => /^OpenRouter V/.test(one.value))) {
      await typeName(input, 'Staging')
    }
    expect(textInputs(openrouter).filter((one) => one.value === 'Staging')).toHaveLength(2)
    await click(findButtons(/discard/i, openrouter)[0] as HTMLElement)
    expect(textInputs(openrouter).filter((one) => one.value === 'Staging')).toHaveLength(1)
    expect(findButtons(/discard/i, openrouter)).toHaveLength(1)
    dispose(wrapper)
  })
})

describe('the name of a new secret', () => {
  it('is the name of the service in a group made and left empty', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      'Default',
      'made-up-llama-value',
    )
    dispose(wrapper)
  })

  it('is followed by V1 in a group that has the first', async () => {
    const { installation, wrapper } = mountBare([secret('s00000001', 'Default', 'Local Llama')])
    await addSecret()
    await enterText(lastMasked(), 'made-up-second-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledWith(
      'Local Llama',
      'Default V1',
      'made-up-second-value',
    )
    dispose(wrapper)
  })

  it('is followed by V2 in a group that has the first and V1', async () => {
    const { installation, wrapper } = mountBare([
      secret('s00000001', 'Local Llama', 'Local Llama'),
      secret('s00000002', 'Default V1', 'Local Llama'),
    ])
    await addSecret()
    await enterText(lastMasked(), 'made-up-third-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledWith(
      'Local Llama',
      'Default V2',
      'made-up-third-value',
    )
    dispose(wrapper)
  })
})

describe('a secret that fails to be created', () => {
  const failing = () => vi.fn(async () => '')

  it('keeps the group and the typed value in the field of an empty group', async () => {
    const createSecret = failing()
    const { wrapper } = mountBare([], { createSecret })
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')

    expect(createSecret).toHaveBeenCalled()
    expect(findText('Local Llama')?.tagName).toBe('H2')
    expect((maskedInputs()[0] as HTMLInputElement).value).toBe('made-up-llama-value')
    dispose(wrapper)
  })

  it('keeps the typed value in the field of a group with secrets', async () => {
    const createSecret = failing()
    const { wrapper } = mountBare([secret('s00000001', 'Local Llama', 'Local Llama')], {
      createSecret,
    })
    await addSecret()
    await enterText(lastMasked(), 'made-up-section-value')

    expect(createSecret).toHaveBeenCalledWith('Local Llama', 'Default V1', 'made-up-section-value')
    expect(lastMasked().value).toBe('made-up-section-value')
    dispose(wrapper)
  })

  it('marks the field invalid and says the value is not saved', async () => {
    const { wrapper } = mountBare([], { createSecret: failing() })
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    const input = maskedInputs()[0] as HTMLInputElement
    expect(input.getAttribute('aria-invalid')).toBe('false')
    await enterText(input, 'made-up-llama-value')

    expect(input.getAttribute('aria-invalid')).toBe('true')
    expect(getDescription(input)).toContain('Not saved. Press Enter to try again.')
    dispose(wrapper)
  })

  it('retries on Enter and clears the mark once it is saved', async () => {
    const stored = ref<ReturnType<typeof secret>[]>([])
    const createSecret = vi
      .fn()
      .mockResolvedValueOnce('')
      .mockImplementation(async (group: string, name: string) => {
        stored.value = [secret('s00000009', name, group)]
        return 's00000009'
      })
    const { wrapper } = mountBare([], { createSecret, ...liveAnswer(stored) })
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    const input = maskedInputs()[0] as HTMLInputElement
    await enterText(input, 'made-up-llama-value')
    expect(input.getAttribute('aria-invalid')).toBe('true')
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()

    expect(createSecret).toHaveBeenCalledTimes(2)
    expect(maskedInputs().every((one) => one.getAttribute('aria-invalid') === 'false')).toBe(true)
    expect(maskedInputs().every((one) => one.value === '')).toBe(true)
    dispose(wrapper)
  })
})

describe('adding a group', () => {
  it('offers to add a group on a page with none and says nothing else', () => {
    const { wrapper } = mountBare()
    expect(findText(/no secrets yet/i)).toBeNull()
    expect(findButton(/add group/i)).toBeTruthy()
    dispose(wrapper)
  })

  it('creates the group at once with a name to type over, focused and selected', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    const input = groupField()
    expect(input.value).toBe('New group')
    expect(document.activeElement).toBe(input)
    expect(input.selectionStart).toBe(0)
    expect(input.selectionEnd).toBe('New group'.length)
    expect(installation.secrets.edit.createSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('numbers the name of a new group when it is taken', async () => {
    const { wrapper } = mountBare([secret('s00000001', 'Work', 'New group')])
    await click(findButton(/add group/i))
    expect(groupField().value).toBe('New group 2')
    await click(findButton(/add group/i))
    expect(findText('New group')).not.toBeNull()
    expect(findText('New group 2')).not.toBeNull()
    dispose(wrapper)
  })

  it('keeps an empty group the person named until a secret is added, then creates the secret in it', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('Local Llama')
    expect(installation.secrets.edit.renameSecretGroup).not.toHaveBeenCalled()
    expect(findText('Local Llama')?.tagName).toBe('H2')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-llama-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledExactlyOnceWith(
      'Local Llama',
      'Default',
      'made-up-llama-value',
    )
    dispose(wrapper)
  })

  it('drops an empty group the person deletes, without a question, and focuses Add group', async () => {
    const { wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('New group')
    await chooseGroupAction(/delete group/i)
    expect(findText('New group')).toBeNull()
    expect(root().querySelector('[role="alertdialog"]')).toBeNull()
    expect(document.activeElement).toBe(findButton(/add group/i))
    dispose(wrapper)
  })
})

describe('the name of a group', () => {
  it('is renamed on the core for a group the person made, and not for a group a plugin asks for', async () => {
    const { installation, wrapper } = mountPage()
    expect(findButtons(/group actions/i, sectionOf('OpenRouter'))).toHaveLength(0)

    const travel = sectionOf('travel')
    await chooseGroupAction(/rename group/i, travel)
    await enterText(travel.querySelector('input[type="text"]') as HTMLInputElement, 'holidays')
    expect(installation.secrets.edit.renameSecretGroup).toHaveBeenCalledExactlyOnceWith(
      'travel',
      'holidays',
    )
    dispose(wrapper)
  })

  it('is refused above 64 characters, and the name stays', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('g'.repeat(65))
    expect(findText('New group')?.tagName).toBe('H2')
    expect(installation.secrets.edit.createSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  it('is accepted at 64 characters', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('g'.repeat(64))
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledWith(
      'g'.repeat(64),
      'Default',
      'made-up-value',
    )
    dispose(wrapper)
  })

  it('is refused where another group of the page has it, with an alert', async () => {
    const { installation, wrapper } = mountPage()
    const travel = sectionOf('travel')
    await chooseGroupAction(/rename group/i, travel)
    await enterText(travel.querySelector('input[type="text"]') as HTMLInputElement, 'openrouter')
    expect(installation.secrets.edit.renameSecretGroup).not.toHaveBeenCalled()
    expect(travel.querySelector('[role="alert"]')?.textContent?.trim()).toBe(
      'A group named openrouter already exists.',
    )
    dispose(wrapper)
  })

  it('is refused where an empty group the person made has it', async () => {
    const { installation, wrapper } = mountBare([MINE])
    await click(findButton(/add group/i))
    await typeGroup('travel')
    expect(installation.secrets.edit.renameSecretGroup).not.toHaveBeenCalled()
    expect(root().querySelector('[role="alert"]')?.textContent).toContain('travel')
    dispose(wrapper)
  })

  it('loses U+0085 at its ends, as the core trims it', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('\u0085abc')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledWith(
      'abc',
      'Default',
      'made-up-value',
    )
    dispose(wrapper)
  })

  it('keeps U+FEFF at its ends, as the core does', async () => {
    const { installation, wrapper } = mountBare()
    await click(findButton(/add group/i))
    await typeGroup('﻿abc')
    await enterText(maskedInputs()[0] as HTMLInputElement, 'made-up-value')
    expect(installation.secrets.edit.createSecret).toHaveBeenCalledWith(
      '﻿abc',
      'Default',
      'made-up-value',
    )
    dispose(wrapper)
  })
})
