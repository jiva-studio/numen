/**
 * The page of secrets in the settings tab, opened by the id `secrets` as the shared settings are
 * opened by `system`. The secrets, the plugins and every value here are invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import SettingsTab from '../SettingsTab.vue'
import { createInstallation, createPluginInfo } from '@/testing/installation'
import {
  chooseGroupAction,
  click,
  dispose,
  enterText,
  findButton,
  findButtons,
  chooseRowAction,
  findMenuItem,
  openRowMenu,
  findNameInput,
  findText,
  maskedInputs,
  nameOf,
  positionOf,
  root,
  rowOf,
  sectionOf,
  typeName,
} from '@/testing/secretsDom'

const EMBEDDINGS = 'md.numen.sample.embeddings'
const PROOFREADING = 'md.numen.sample.proofreading'

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
  const installation = createInstallation({
    plugins: ref(plugins),
    secrets: ref(secrets),
    ...overrides,
  })
  const wrapper = mount(SettingsTab, {
    attachTo: document.body,
    props: { state: { installation, initialPluginId: ref('secrets') } },
  })
  return { installation, wrapper }
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

describe('the page of secrets', () => {
  it('has a section for each group, the groups plugins ask for by name and then the ones the person made by name', () => {
    const { wrapper } = mountPage()
    const order = ['OpenAI', 'OpenRouter', 'travel'].map((group) => findText(group))
    expect(order.every((one) => one !== null)).toBe(true)
    const positions = order.map((one) => positionOf(one as HTMLElement))
    expect(positions).toStrictEqual([...positions].sort((a, b) => a - b))
    dispose(wrapper)
  })

  it('lists the secrets of a group in priority order', () => {
    const { wrapper } = mountPage()
    expect(positionOf(findNameInput('Work') as HTMLElement)).toBeLessThan(
      positionOf(findNameInput('Home') as HTMLElement),
    )
    dispose(wrapper)
  })

  it('shows a group a plugin asks for with no secret as one empty secret named after its service', () => {
    const { wrapper } = mountPage()
    const openai = findText('OpenAI') as HTMLElement
    const fallback = rowOf('OpenAI')
    expect(fallback).not.toBeNull()
    expect(positionOf(fallback)).toBeGreaterThan(positionOf(openai))
    expect(positionOf(fallback)).toBeLessThan(positionOf(findText('OpenRouter') as HTMLElement))
    dispose(wrapper)
  })

  it('stores the empty secret under its group on the first entry of a value', async () => {
    const { installation, wrapper } = mountPage()
    const input = maskedInputs(rowOf('OpenAI'))[0] as HTMLInputElement
    await enterText(input, 'made-up-first-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openai',
      'OpenAI',
      'made-up-first-value',
    )
    dispose(wrapper)
  })

  it('moves a secret up before the one above it and down after the one below it', async () => {
    const { installation, wrapper } = mountPage()
    const home = rowOf('Home')
    await chooseRowAction(home, /move up/i)
    expect(installation.moveSecret).toHaveBeenLastCalledWith(HOME.id, WORK.id)

    const work = rowOf('Work')
    await chooseRowAction(work, /move down/i)
    expect(installation.moveSecret).toHaveBeenLastCalledWith(WORK.id, '')
    dispose(wrapper)
  })

  it('cannot move the first secret of a group up or the last one down', async () => {
    const first = mountPage()
    const itemsOfWork = await openRowMenu(rowOf('Work'))
    const up = itemsOfWork.find((one) => /move up/i.test(nameOf(one)))
    expect(up?.hasAttribute('disabled')).toBe(true)
    dispose(first.wrapper)

    const second = mountPage()
    const itemsOfHome = await openRowMenu(rowOf('Home'))
    const down = itemsOfHome.find((one) => /move down/i.test(nameOf(one)))
    expect(down?.hasAttribute('disabled')).toBe(true)
    dispose(second.wrapper)
  })

  it('renames a secret by its id', async () => {
    const { installation, wrapper } = mountPage()
    const work = rowOf('Work')
    const input = findNameInput('Work') as HTMLInputElement
    expect(findButtons(/rename/i, work)).toHaveLength(0)
    await typeName(input, 'Office')
    expect(installation.renameSecret).toHaveBeenCalledExactlyOnceWith(WORK.id, 'Office')
    dispose(wrapper)
  })

  it('asks for a confirmation that names the plugins that can read the secret before it deletes one', async () => {
    const { installation, wrapper } = mountPage()
    const work = rowOf('Work')
    await chooseRowAction(work, /^delete$/i)

    expect(installation.deleteSecret).not.toHaveBeenCalled()
    const dialog = root().querySelector(
      '[role="dialog"], [role="alertdialog"], dialog',
    ) as HTMLElement
    expect(dialog.textContent).toContain('Embeddings')
    expect(dialog.textContent).not.toContain('Proofreading')

    await click(findButton(/delete|confirm/i, dialog))
    expect(installation.deleteSecret).toHaveBeenCalledExactlyOnceWith(WORK.id)
    dispose(wrapper)
  })

  it('deletes nothing when the confirmation is cancelled', async () => {
    const { installation, wrapper } = mountPage()
    const work = rowOf('Work')
    await chooseRowAction(work, /^delete$/i)
    const dialog = root().querySelector(
      '[role="dialog"], [role="alertdialog"], dialog',
    ) as HTMLElement
    await click(findButton(/cancel/i, dialog))
    expect(installation.deleteSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  const openReaders = (row: HTMLElement) => click(findButton(/plugin/i, row))
  const optionOf = (pluginName: string) =>
    Array.from(root().querySelectorAll<HTMLElement>('[role="option"]')).find(
      (one) => one.textContent?.trim() === pluginName,
    ) as HTMLElement
  const isMarked = (option: HTMLElement) =>
    option.querySelector('.secret-permissions__check--off') === null
  const optionNames = () =>
    Array.from(root().querySelectorAll('[role="option"]')).map((one) => one.textContent?.trim())

  it('lists a plugin once per name: its declarations of the group, and the ungrouped one only where it has none for the group', async () => {
    const { wrapper } = mountPage()
    await openReaders(rowOf('Work'))
    expect(optionNames()).toStrictEqual(['Embeddings', 'Proofreading'])
    dispose(wrapper)
  })

  it('lists the declarations of the other groups in no section', async () => {
    const { wrapper } = mountPage()
    await openReaders(rowOf('Personal'))
    expect(optionNames()).toStrictEqual(['Embeddings'])
    dispose(wrapper)
  })

  it('reads the number of plugins that may read the secret on the button', () => {
    const { wrapper } = mountPage()
    expect(findButton(/plugin/i, rowOf('Work')).textContent).toContain('1 plugin')
    expect(findButton(/plugin/i, rowOf('Home')).textContent).toContain('No plugins')
    dispose(wrapper)
  })

  it('marks an allowed plugin with a check and toggles it with allow and deny', async () => {
    const { installation, wrapper } = mountPage()
    await openReaders(rowOf('Work'))
    const embeddingsBox = optionOf('Embeddings')
    const proofreadingBox = optionOf('Proofreading')
    expect(isMarked(embeddingsBox)).toBe(true)
    expect(isMarked(proofreadingBox)).toBe(false)

    await click(proofreadingBox)
    expect(installation.allowPluginSecret).toHaveBeenLastCalledWith(
      WORK.id,
      PROOFREADING,
      'apiKey',
      'openrouter',
    )
    await click(embeddingsBox)
    expect(installation.denyPluginSecret).toHaveBeenLastCalledWith(
      WORK.id,
      EMBEDDINGS,
      'apiKey',
      'openrouter',
    )
    dispose(wrapper)
  })

  it('offers the order items only in a group with more than one secret', async () => {
    const first = mountPage()
    const inWork = await openRowMenu(rowOf('Work'))
    expect(inWork.map(nameOf)).toEqual(['Move up', 'Move down', 'Delete'])
    dispose(first.wrapper)

    const second = mountPage()
    const inPersonal = await openRowMenu(rowOf('Personal'))
    expect(inPersonal.map(nameOf)).toEqual(['Delete'])
    dispose(second.wrapper)
  })

  it('adds a secret with the next version as its name and stores it when a value is entered', async () => {
    const { installation, wrapper } = mountPage()
    const openrouter = sectionOf('OpenRouter')
    await chooseGroupAction(/add secret/i, openrouter)
    expect(findNameInput('OpenRouter V1')).not.toBeNull()
    await enterText(
      maskedInputs(rowOf('OpenRouter V1'))[0] as HTMLInputElement,
      'made-up-added-value',
    )
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openrouter',
      'OpenRouter V1',
      'made-up-added-value',
    )
    dispose(wrapper)
  })

  it('creates a new secret under the name typed into its name input', async () => {
    const { installation, wrapper } = mountPage()
    await chooseGroupAction(/add secret/i, sectionOf('OpenRouter'))
    await typeName(findNameInput('OpenRouter V1') as HTMLInputElement, 'Staging')
    await enterText(maskedInputs(rowOf('Staging'))[0] as HTMLInputElement, 'made-up-staging-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openrouter',
      'Staging',
      'made-up-staging-value',
    )
    dispose(wrapper)
  })

  it('draws a new secret like a saved one, with its plugin list disabled until it exists', async () => {
    const { wrapper } = mountPage()
    await chooseGroupAction(/add secret/i, sectionOf('OpenRouter'))
    const draft = rowOf('OpenRouter V1')
    const saved = rowOf('Work')
    expect(draft.className).toBe(saved.className)
    expect(findButton(/save a value first/i, draft).hasAttribute('disabled')).toBe(true)
    expect(findButton(/save a value first/i, draft).title).toBe('Save a value first')
    const discard = findButton(/discard/i, draft)
    const menu = findButton(/secret actions/i, saved)
    expect(discard.className).toBe(menu.className)
    expect(discard.querySelector('svg')?.getAttribute('width')).toBe(
      menu.querySelector('svg')?.getAttribute('width'),
    )
    dispose(wrapper)
  })

  it('discards a new secret with its button', async () => {
    const { wrapper } = mountPage()
    await chooseGroupAction(/add secret/i, sectionOf('OpenRouter'))
    await click(findButton(/discard/i, rowOf('OpenRouter V1')))
    expect(findNameInput('OpenRouter V1')).toBeNull()
    dispose(wrapper)
  })

  it('names the second secret of a group with V1', async () => {
    const { wrapper } = mountPage()
    const travel = sectionOf('travel')
    await chooseGroupAction(/add secret/i, travel)
    expect(findNameInput('travel V1')).not.toBeNull()
    dispose(wrapper)
  })

  it('renames a group the person made and not a group a plugin asks for', async () => {
    const { installation, wrapper } = mountPage()
    const travel = sectionOf('travel')
    const openrouter = sectionOf('OpenRouter')
    await click(findButton(/group actions/i, openrouter))
    expect(root().querySelectorAll('[role="menuitem"]')).toHaveLength(1)
    expect(findMenuItem(/add secret/i)).toBeTruthy()
    await click(findMenuItem(/add secret/i))

    await chooseGroupAction(/rename group/i, travel)
    const input = travel.querySelector('input[type="text"], input:not([type])') as HTMLInputElement
    await enterText(input, 'holidays')
    expect(installation.renameSecretGroup).toHaveBeenCalledExactlyOnceWith('travel', 'holidays')
    dispose(wrapper)
  })

  it('shows the path of the file only when the fallback file is in use', () => {
    const path = '/home/someone/.config/numen/credentials/credentials.json'
    const { wrapper } = mountPage([WORK], { credentialsPath: ref(path) })
    expect(root().textContent).toContain(path)
    dispose(wrapper)
    const { wrapper: other } = mountPage([WORK], { credentialsPath: ref('') })
    expect(root().textContent).not.toContain('credentials')
    dispose(other)
  })

  it('always shows the field, with dots as its placeholder while a value is stored', () => {
    const { wrapper } = mountPage([WORK])
    const input = maskedInputs(rowOf('Work'))[0] as HTMLInputElement
    expect(input.placeholder).toBe('••••••••')
    expect(input.value).toBe('')
    dispose(wrapper)
  })

  it('writes a value entered over a stored one', async () => {
    const { installation, wrapper } = mountPage([WORK])
    const input = maskedInputs(rowOf('Work'))[0] as HTMLInputElement
    await enterText(input, 'made-up-new-value')
    expect(installation.writeSecret).toHaveBeenCalledExactlyOnceWith(WORK.id, 'made-up-new-value')
    dispose(wrapper)
  })

  it('shows an empty field with a placeholder where no value is stored', () => {
    const { wrapper } = mountPage([secret('s00000009', 'Imported', 'openrouter', [], false)])
    const row = rowOf('Imported')
    expect(maskedInputs(row)[0]?.placeholder).toBe('Paste a value')
    expect(maskedInputs(row)[0]?.placeholder).not.toBe('••••••••')
    dispose(wrapper)
  })

  it('shows no value after it is entered', async () => {
    const { installation, wrapper } = mountPage()
    const input = maskedInputs(rowOf('OpenAI'))[0] as HTMLInputElement
    await enterText(input, 'made-up-value-that-must-not-stay')

    expect(installation.createSecret).toHaveBeenCalledOnce()
    expect(input.value).toBe('')
    expect(root().innerHTML).not.toContain('made-up-value-that-must-not-stay')
    for (const field of Array.from(root().querySelectorAll('input, textarea'))) {
      expect((field as HTMLInputElement).value).not.toContain('made-up-value-that-must-not-stay')
    }
    dispose(wrapper)
  })
})
