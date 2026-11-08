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
  click,
  dispose,
  enterText,
  findButton,
  findButtons,
  findText,
  maskedInputs,
  positionOf,
  root,
  rowOf,
  sectionOf,
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
    const order = ['openai', 'openrouter', 'travel'].map((group) => findText(group))
    expect(order.every((one) => one !== null)).toBe(true)
    const positions = order.map((one) => positionOf(one as HTMLElement))
    expect(positions).toStrictEqual([...positions].sort((a, b) => a - b))
    dispose(wrapper)
  })

  it('lists the secrets of a group in priority order', () => {
    const { wrapper } = mountPage()
    expect(positionOf(findText('Work') as HTMLElement)).toBeLessThan(
      positionOf(findText('Home') as HTMLElement),
    )
    dispose(wrapper)
  })

  it('shows a group a plugin asks for with no secret as one empty secret named Default', () => {
    const { wrapper } = mountPage()
    const openai = findText('openai') as HTMLElement
    const fallback = findText('Default') as HTMLElement
    expect(fallback).not.toBeNull()
    expect(positionOf(fallback)).toBeGreaterThan(positionOf(openai))
    expect(positionOf(fallback)).toBeLessThan(positionOf(findText('openrouter') as HTMLElement))
    dispose(wrapper)
  })

  it('stores the empty secret under its group on the first entry of a value', async () => {
    const { installation, wrapper } = mountPage()
    const input = maskedInputs(rowOf('Default'))[0] as HTMLInputElement
    await enterText(input, 'made-up-first-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openai',
      'Default',
      'made-up-first-value',
    )
    dispose(wrapper)
  })

  it('moves a secret up before the one above it and down after the one below it', async () => {
    const { installation, wrapper } = mountPage()
    const home = rowOf('Home')
    await click(findButton(/move up/i, home))
    expect(installation.moveSecret).toHaveBeenLastCalledWith(HOME.id, WORK.id)

    const work = rowOf('Work')
    await click(findButton(/move down/i, work))
    expect(installation.moveSecret).toHaveBeenLastCalledWith(WORK.id, '')
    dispose(wrapper)
  })

  it('cannot move the first secret of a group up or the last one down', () => {
    const { wrapper } = mountPage()
    const work = rowOf('Work')
    const home = rowOf('Home')
    expect(findButton(/move up/i, work).hasAttribute('disabled')).toBe(true)
    expect(findButton(/move down/i, home).hasAttribute('disabled')).toBe(true)
    dispose(wrapper)
  })

  it('renames a secret by its id', async () => {
    const { installation, wrapper } = mountPage()
    const work = rowOf('Work')
    await click(findButton(/rename/i, work))
    const input = root().querySelector<HTMLInputElement>('input[type="text"], input:not([type])')
    expect(input?.value).toBe('Work')
    await enterText(input as HTMLInputElement, 'Office')
    expect(installation.renameSecret).toHaveBeenCalledExactlyOnceWith(WORK.id, 'Office')
    dispose(wrapper)
  })

  it('asks for a confirmation that names the plugins that can read the secret before it deletes one', async () => {
    const { installation, wrapper } = mountPage()
    const work = rowOf('Work')
    await click(findButton(/delete/i, work))

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
    await click(findButton(/delete/i, work))
    const dialog = root().querySelector(
      '[role="dialog"], [role="alertdialog"], dialog',
    ) as HTMLElement
    await click(findButton(/cancel/i, dialog))
    expect(installation.deleteSecret).not.toHaveBeenCalled()
    dispose(wrapper)
  })

  const openReaders = (row: HTMLElement) => click(findButton(/plugin/i, row))
  const checkboxOf = (description: string) =>
    root().querySelector<HTMLInputElement>(
      `label[title="${description}"] input`,
    ) as HTMLInputElement

  it('lists in a section only the declarations of its group and those with no group', async () => {
    const { wrapper } = mountPage()
    await openReaders(rowOf('Work'))
    const titles = Array.from(root().querySelectorAll('label')).map((one) => one.title)
    expect(titles).toStrictEqual([
      'Key of the embeddings service',
      'Key of the custom embeddings service',
      'Key of the proofreading service',
    ])
    expect(root().querySelectorAll('input[type="checkbox"]')).toHaveLength(3)
    dispose(wrapper)
  })

  it('lists the declarations of the other groups in no section', async () => {
    const { wrapper } = mountPage()
    await openReaders(rowOf('Personal'))
    const titles = Array.from(root().querySelectorAll('label')).map((one) => one.title)
    expect(titles).toStrictEqual(['Key of the custom embeddings service'])
    dispose(wrapper)
  })

  it('reads the number of plugins that may read the secret on the button', () => {
    const { wrapper } = mountPage()
    expect(findButton(/plugin/i, rowOf('Work')).textContent).toContain('1 plugin')
    expect(findButton(/plugin/i, rowOf('Home')).textContent).toContain('No plugins')
    dispose(wrapper)
  })

  it('shows a permission as checked and toggles it with allow and deny', async () => {
    const { installation, wrapper } = mountPage()
    await openReaders(rowOf('Work'))
    const embeddingsBox = checkboxOf('Key of the embeddings service')
    const proofreadingBox = checkboxOf('Key of the proofreading service')
    expect(embeddingsBox.checked).toBe(true)
    expect(proofreadingBox.checked).toBe(false)

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

  it('shows order arrows only in a group with more than one secret', () => {
    const { wrapper } = mountPage()
    expect(findButtons(/move up/i, rowOf('Work'))).toHaveLength(1)
    expect(findButtons(/move up|move down/i, rowOf('Personal'))).toHaveLength(0)
    dispose(wrapper)
  })

  it('adds a secret with the next number as its name and stores it when a value is entered', async () => {
    const { installation, wrapper } = mountPage()
    const openrouter = sectionOf('openrouter')
    await click(findButton(/add/i, openrouter))
    const draft = findText('3') as HTMLElement
    expect(draft).not.toBeNull()
    await enterText(maskedInputs(rowOf('3'))[0] as HTMLInputElement, 'made-up-third-value')
    expect(installation.createSecret).toHaveBeenCalledExactlyOnceWith(
      'openrouter',
      '3',
      'made-up-third-value',
    )
    dispose(wrapper)
  })

  it('names the second secret of a group 2', async () => {
    const { wrapper } = mountPage()
    const travel = sectionOf('travel')
    await click(findButton(/add/i, travel))
    expect(findText('2')).not.toBeNull()
    dispose(wrapper)
  })

  it('renames a group the person made and not a group a plugin asks for', async () => {
    const { installation, wrapper } = mountPage()
    const travel = sectionOf('travel')
    const openrouter = sectionOf('openrouter')
    expect(findButtons(/rename group/i, openrouter)).toHaveLength(0)

    await click(findButton(/rename group/i, travel))
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

  it('shows a stored value as dots with a Replace action and no field', () => {
    const { wrapper } = mountPage([WORK])
    const row = rowOf('Work')
    expect(row.textContent).toContain('••••••••')
    expect(maskedInputs(row)).toHaveLength(0)
    expect(findButton(/replace/i, row)).toBeTruthy()
    dispose(wrapper)
  })

  it('opens an empty field on Replace and writes the value entered', async () => {
    const { installation, wrapper } = mountPage([WORK])
    await click(findButton(/replace/i, rowOf('Work')))
    const input = maskedInputs(rowOf('Work'))[0] as HTMLInputElement
    expect(input.value).toBe('')
    expect(document.activeElement).toBe(input)
    await enterText(input, 'made-up-new-value')
    expect(installation.writeSecret).toHaveBeenCalledExactlyOnceWith(WORK.id, 'made-up-new-value')
    dispose(wrapper)
  })

  it('shows an empty field with a placeholder where no value is stored', () => {
    const { wrapper } = mountPage([secret('s00000009', 'Imported', 'openrouter', [], false)])
    const row = rowOf('Imported')
    expect(maskedInputs(row)[0]?.placeholder).toBe('Paste a value')
    expect(row.textContent).not.toContain('••••••••')
    dispose(wrapper)
  })

  it('shows no value after it is entered', async () => {
    const { installation, wrapper } = mountPage()
    const input = maskedInputs(rowOf('Default'))[0] as HTMLInputElement
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
