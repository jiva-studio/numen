/**
 * The page "Identity and licenses" inside the settings tab: a person sets up an identity, checks
 * three of its twelve words, restores it from words and deletes it from this device. The core is
 * replaced by the calls of the installation; the page, its model and the settings tab are real.
 *
 * Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { asFailure, asValue, type Result } from '@numen/result'
import type { IdentityInfo } from '@/entities/identity'
import type { LicenseInfo } from '@/entities/license'
import {
  createInertDevices,
  createInstallation,
  createSettingsState,
} from '@/pages/settings/testing/installation'
import SettingsTab from '../SettingsTab.vue'
import { WORDS as words } from '../../words'

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

const IDENTITY: IdentityInfo = {
  accountFingerprint: '3F2A 91C0 0000 1111 2222 3333 4444 5555',
  deviceFingerprint: 'AAAA BBBB CCCC DDDD 0000 1111 2222 3333',
  deviceName: 'Laptop',
  deviceAddedAt: '2026-11-22T12:00:00Z',
  isRevoked: false,
  keyEpoch: 0,
}

const SYNC: LicenseInfo = {
  id: 'numen.sync',
  title: 'Numen Sync',
  contributors: [],
  providers: [],
  status: 'inactive',
  isActive: false,
  hasKey: false,
  keyHint: '',
  activeProvider: '',
  instanceName: '',
  expiresAt: '',
  verifiedAt: '',
}

const mounted: VueWrapper[] = []

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
  while (mounted.length) mounted.pop()?.unmount()
})

const createPage = async (
  initial: IdentityInfo | undefined = undefined,
  create: (words: readonly string[]) => Promise<Result<IdentityInfo, string>> = () =>
    Promise.resolve(asValue(IDENTITY)),
) => {
  const identity: Ref<IdentityInfo | undefined> = ref(initial)
  const calls = {
    load: vi.fn(() => Promise.resolve()),
    generateWords: vi.fn(() => Promise.resolve(asValue<readonly string[]>(RECOVERY_WORDS))),
    create: vi.fn(async (given: readonly string[]) => {
      const answer = await create(given)
      if (answer.ok) identity.value = IDENTITY
      return answer
    }),
    restore: vi.fn(async (given: readonly string[]) => {
      const answer = await create(given)
      if (answer.ok) identity.value = IDENTITY
      return answer
    }),
    delete: vi.fn(() => {
      identity.value = undefined
      return Promise.resolve(asValue<void>(undefined))
    }),
  }
  const installation = createInstallation({
    licenses: {
      list: ref([SYNC]),
      load: vi.fn(() => Promise.resolve()),
      watch: vi.fn(() => Promise.resolve()),
      activate: vi.fn(() => Promise.resolve(asFailure('Not activated.'))),
      deactivate: vi.fn(() => Promise.resolve(asFailure('Not deactivated.'))),
    },
    identity: { identity, ...calls, devices: createInertDevices() },
  })
  const tab = mount(SettingsTab, {
    props: { state: createSettingsState(installation, { initialSearchQuery: ref('') }) },
    attachTo: document.body,
  })
  mounted.push(tab)
  await tab.get('button[aria-label="Identity and licenses"]').trigger('click')
  await flushPromises()
  return { tab, identity, ...calls }
}

type Page = Awaited<ReturnType<typeof createPage>>['tab']

const findButton = (tab: Page, text: string) =>
  tab.findAll('button').find((one) => one.text() === text)

const press = async (tab: Page, text: string) => {
  const button = findButton(tab, text)
  if (!button) throw new Error(`the page has no button "${text}"`)
  await button.trigger('click')
  await flushPromises()
}

const getShownWords = (tab: Page): string[] =>
  tab.findAll('ol[aria-label="Recovery words"] li').map((one) => one.text())

const getConfirmInputs = (tab: Page) => tab.findAll('input[aria-label^="Confirm word "]')

const getPosition = (input: { attributes: (name: string) => string | undefined }): number =>
  Number(/^Confirm word (\d+)$/.exec(input.attributes('aria-label') ?? '')?.[1])

const fillConfirmations = async (tab: Page, mistake?: { word: string }) => {
  const inputs = getConfirmInputs(tab)
  for (const [index, input] of inputs.entries()) {
    const word = RECOVERY_WORDS[getPosition(input) - 1] ?? ''
    await input.setValue(index === 0 && mistake ? mistake.word : word)
  }
}

describe('the page "Identity and licenses"', () => {
  it('opens from the settings list and keeps the license cards on the same page', async () => {
    const { tab } = await createPage()
    expect(tab.get('[aria-current="page"]').text()).toBe(words.identityAndLicenses)
    expect(tab.text()).toContain('Numen Sync')
  })

  it('loads the identity of this device when it opens', async () => {
    const { load } = await createPage()
    expect(load).toHaveBeenCalled()
  })

  it('offers to create or restore an identity when the device has none', async () => {
    const { tab } = await createPage()
    expect(tab.text()).toContain('No identity on this device')
    expect(findButton(tab, 'Create identity')).toBeDefined()
    expect(findButton(tab, 'Restore from words')).toBeDefined()
    expect(tab.text()).not.toContain('Account fingerprint')
  })

  it('shows the account fingerprint and this device when the device has an identity', async () => {
    const { tab } = await createPage(IDENTITY)
    expect(tab.text()).toContain('Account fingerprint')
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.text()).toContain('This device')
    expect(tab.text()).toContain('Laptop')
    expect(tab.text()).toContain(IDENTITY.deviceFingerprint)
    expect(findButton(tab, 'Create identity')).toBeUndefined()
  })

  it('shows twelve recovery words and asks for the check of three of them', async () => {
    const { tab, generateWords } = await createPage()
    await press(tab, 'Create identity')
    expect(generateWords).toHaveBeenCalledOnce()
    expect(getShownWords(tab)).toStrictEqual(RECOVERY_WORDS)
    expect(tab.text()).toContain('Write down these 12 words. They are shown only now.')
    const positions = getConfirmInputs(tab).map(getPosition)
    expect(positions).toHaveLength(3)
    expect(new Set(positions).size).toBe(3)
    expect(positions.every((position) => position >= 1 && position <= 12)).toBe(true)
  })

  it('keeps the identity once the three words are right and shows no word again', async () => {
    const { tab, create } = await createPage()
    await press(tab, 'Create identity')
    await fillConfirmations(tab)
    await press(tab, 'Keep identity')
    expect(create).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.find('ol[aria-label="Recovery words"]').exists()).toBe(false)
    expect(getConfirmInputs(tab)).toHaveLength(0)
    for (const word of RECOVERY_WORDS) expect(tab.text()).not.toContain(word)
  })

  it('calls nothing and says the words are not valid when a confirmed word is wrong', async () => {
    const { tab, create } = await createPage()
    await press(tab, 'Create identity')
    await fillConfirmations(tab, { word: 'zoo' })
    await press(tab, 'Keep identity')
    expect(create).not.toHaveBeenCalled()
    expect(tab.get('[role="alert"]').text()).toBe('Some words are not valid.')
    expect(tab.text()).toContain('No identity on this device')
  })

  it('draws the words again after a wrong confirmation so the person can look at them', async () => {
    const { tab } = await createPage()
    await press(tab, 'Create identity')
    await fillConfirmations(tab, { word: 'zoo' })
    await press(tab, 'Keep identity')
    expect(getShownWords(tab)).toStrictEqual(RECOVERY_WORDS)
  })

  it('drops the words from the page when the person cancels', async () => {
    const { tab, create } = await createPage()
    await press(tab, 'Create identity')
    await press(tab, 'Cancel')
    expect(create).not.toHaveBeenCalled()
    for (const word of RECOVERY_WORDS) expect(tab.text()).not.toContain(word)
    expect(findButton(tab, 'Create identity')).toBeDefined()
  })

  it('asks for new words each time a setup starts', async () => {
    const { tab, generateWords } = await createPage()
    await press(tab, 'Create identity')
    await press(tab, 'Cancel')
    await press(tab, 'Create identity')
    expect(generateWords).toHaveBeenCalledTimes(2)
  })

  it('starts a restore with the words that are typed when the person restores', async () => {
    const { tab, create, restore } = await createPage()
    await press(tab, 'Restore from words')
    await tab
      .get('textarea[aria-label="Your 12 recovery words"]')
      .setValue(`  ${RECOVERY_WORDS.join('  ')}\n`)
    await press(tab, 'Restore')
    expect(restore).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(create).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('shows "Restoring…" until the core has the identity', async () => {
    let finish: (answer: Result<IdentityInfo, string>) => void = () => undefined
    const { tab, identity } = await createPage(
      undefined,
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    await press(tab, 'Restore from words')
    await tab
      .get('textarea[aria-label="Your 12 recovery words"]')
      .setValue(RECOVERY_WORDS.join(' '))

    await press(tab, 'Restore')

    expect(findButton(tab, words.identityRestoring)?.attributes('disabled')).toBeDefined()
    expect(identity.value).toBeUndefined()
    expect(tab.text()).not.toContain(IDENTITY.accountFingerprint)
    finish(asValue(IDENTITY))
    await flushPromises()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('says the words are not valid and holds no identity when the core refuses the words', async () => {
    const { tab, identity } = await createPage(undefined, () =>
      Promise.resolve(asFailure('Some words are not valid.')),
    )
    await press(tab, 'Restore from words')
    await tab.get('textarea[aria-label="Your 12 recovery words"]').setValue('abandon '.repeat(12))
    await press(tab, 'Restore')
    expect(tab.get('[role="alert"]').text()).toBe('Some words are not valid.')
    expect(identity.value).toBeUndefined()
    expect(tab.text()).toContain('No identity on this device')
  })

  it('warns that the identity cannot be restored without the words before it deletes', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete from this device')
    await nextTick()
    const dialog = document.body.querySelector('[role="alertdialog"]')
    expect(dialog?.textContent).toContain('Without the 12 words this identity cannot be restored.')
    expect(remove).not.toHaveBeenCalled()
  })

  it('keeps the identity when the person cancels the delete', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete from this device')
    const cancel = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button'),
    ).find((one) => one.textContent?.trim() === 'Cancel')
    cancel?.click()
    await flushPromises()
    expect(remove).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('deletes the identity once the person confirms', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete from this device')
    const confirm = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button'),
    ).find((one) => one.textContent?.trim() === 'Delete')
    confirm?.click()
    await flushPromises()
    expect(remove).toHaveBeenCalledOnce()
    expect(tab.text()).toContain('No identity on this device')
  })
})
