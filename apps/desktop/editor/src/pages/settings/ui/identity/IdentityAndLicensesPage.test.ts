/**
 * The page "Identity and licenses" inside the settings tab: a person sets up an identity in a dialog
 * of two steps (twelve words, then three of them to check), restores it from words or sheets in a
 * dialog, saves a recovery kit as a PDF and deletes it from this device. The core is replaced by the calls of the installation; the page, its model and the settings tab are real.
 *
 * Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { asFailure, asValue, type Result } from '@numen/result'
import type { IdentityInfo } from '@/entities/identity'
import type { LicenseInfo } from '@/entities/license'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'
import SettingsTab from '../SettingsTab.vue'
import { WORDS as words } from '../../words'

const NO_IDENTITY = 'An identity is your Numen account.'

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
  expiresUnixMs: 0,
  verifiedUnixMs: 0,
}

const createSheetWords = (sheet: number): string[] =>
  Array.from({ length: 33 }, (_, index) => `s${sheet}w${index + 1}`)

type KitProof = { words: readonly string[] } | { sheets: readonly (readonly string[])[] }
type Sheets = readonly (readonly string[])[]

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
  kitAnswers: {
    saveKit?: Result<boolean, string>
    restoreFromKit?: Result<IdentityInfo, string>
  } = {},
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
  const kitCalls = {
    saveKit: vi.fn((_proof: KitProof) => Promise.resolve(kitAnswers.saveKit ?? asValue(true))),
    restoreFromKit: vi.fn((_sheets: Sheets) => {
      const answer = kitAnswers.restoreFromKit ?? asValue(IDENTITY)
      if (answer.ok) identity.value = answer.value
      return Promise.resolve(answer)
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
    identity: { identity, ...calls, ...kitCalls },
  })
  const tab = mount(SettingsTab, {
    props: { state: createSettingsState(installation, { initialSearchQuery: ref('') }) },
    attachTo: document.body,
  })
  mounted.push(tab)
  await tab.get('button[aria-label="Identity and licenses"]').trigger('click')
  await flushPromises()
  return { tab, identity, ...calls, ...kitCalls }
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

const getConfirmInputs = (tab: Page) => tab.findAll('label.word-check input')

const getDialog = (tab: Page) => tab.get('[role="dialog"]')

const getRestoreInputs = (tab: Page) =>
  getDialog(tab).findAll('input[aria-label^="Recovery word "]')

const submitRestore = async (tab: Page) => {
  await getDialog(tab).get('button[type="submit"]').trigger('click')
  await flushPromises()
}

const getPosition = (input: { element: Element }): number =>
  Number(
    /^Word (\d+)$/.exec(
      (input.element as HTMLInputElement).labels?.[0]?.textContent?.trim() ?? '',
    )?.[1],
  )

const startSetup = async (tab: Page) => {
  await press(tab, 'Create identity')
  await press(tab, 'I wrote the words down')
}

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
    expect(tab.text()).toContain(NO_IDENTITY)
    expect(findButton(tab, 'Create identity')).toBeDefined()
    expect(findButton(tab, 'Restore')).toBeDefined()
    expect(tab.text()).not.toContain('Account fingerprint')
  })

  it('shows the account fingerprint and the linked device when the device has an identity', async () => {
    const { tab } = await createPage(IDENTITY)
    expect(tab.text()).toContain('Account fingerprint')
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.text()).toContain('Laptop')
    expect(tab.text()).toContain('This device')
    expect(tab.text()).toContain('Linked November 22, 2026')
    expect(tab.text().match(/November 22, 2026/g)).toHaveLength(1)
    expect(tab.text()).not.toContain('Created')
    expect(tab.text()).toContain('Devices')
    expect(tab.text()).toContain('1 device')
    expect(tab.text()).toContain(IDENTITY.deviceFingerprint)
    expect(findButton(tab, 'Create identity')).toBeUndefined()
  })

  it('shows the twelve numbered recovery words alone in the first step', async () => {
    const { tab, generateWords } = await createPage()
    await press(tab, 'Create identity')
    expect(generateWords).toHaveBeenCalledOnce()
    expect(getShownWords(tab)).toStrictEqual(RECOVERY_WORDS)
    expect(tab.text()).toContain('Write the 12 words down in order. We show them once.')
    expect(tab.text()).toContain('Neither Numen nor support can recover these words')
    expect(getConfirmInputs(tab)).toHaveLength(0)
    expect(findButton(tab, 'Cancel')).toBeDefined()
  })

  it('asks for three labelled words in the second step and shows no word', async () => {
    const { tab } = await createPage()
    await startSetup(tab)
    const positions = getConfirmInputs(tab).map(getPosition)
    expect(positions).toHaveLength(3)
    expect(new Set(positions).size).toBe(3)
    expect(positions.every((position) => position >= 1 && position <= 12)).toBe(true)
    expect(tab.get('[role="dialog"] h2').text()).toBe('Check the words')
    expect(tab.text()).toContain('from your record.')
    expect(getShownWords(tab)).toStrictEqual([])
  })

  it('keeps the identity once the three words are right and shows no word again', async () => {
    const { tab, create } = await createPage()
    await startSetup(tab)
    await fillConfirmations(tab)
    await press(tab, 'Continue')
    expect(create).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.find('ol[aria-label="Recovery words"]').exists()).toBe(false)
    expect(getConfirmInputs(tab)).toHaveLength(0)
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    for (const word of RECOVERY_WORDS) expect(tab.text()).not.toContain(word)
  })

  it('does not continue until all three words are typed', async () => {
    const { tab, create } = await createPage()
    await startSetup(tab)
    await getConfirmInputs(tab)[0]?.setValue('legal')
    const next = findButton(tab, 'Continue')
    expect(next?.attributes('disabled')).toBeDefined()
    expect(create).not.toHaveBeenCalled()
  })

  it('calls nothing and says the words are not valid when a confirmed word is wrong', async () => {
    const { tab, create } = await createPage()
    await startSetup(tab)
    await fillConfirmations(tab, { word: 'zoo' })
    await press(tab, 'Continue')
    expect(create).not.toHaveBeenCalled()
    const alert = tab.get('[role="alert"]')
    expect(alert.text()).toBe('Some words are not valid.')
    expect(tab.text()).toContain(NO_IDENTITY)
    const inputs = getConfirmInputs(tab)
    expect(inputs.map((input) => input.attributes('aria-invalid'))).toStrictEqual([
      'true',
      undefined,
      undefined,
    ])
    const lastInput = inputs.at(-1)?.element
    const back = findButton(tab, 'Back')?.element
    if (!lastInput || !back) throw new Error('the check has no inputs or no Back')
    expect(
      lastInput.compareDocumentPosition(alert.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      alert.element.compareDocumentPosition(back) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('clears the mark of a wrong word once the person types it again', async () => {
    const { tab } = await createPage()
    await startSetup(tab)
    await fillConfirmations(tab, { word: 'zoo' })
    await press(tab, 'Continue')
    const first = getConfirmInputs(tab)[0]
    if (!first) throw new Error('the check has no input')
    await first.setValue(RECOVERY_WORDS[getPosition(first) - 1] ?? '')
    expect(getConfirmInputs(tab)[0]?.attributes('aria-invalid')).toBeUndefined()
  })

  it('goes back to the words from the second step and drops the problem', async () => {
    const { tab } = await createPage()
    await startSetup(tab)
    await fillConfirmations(tab, { word: 'zoo' })
    await press(tab, 'Continue')
    await press(tab, 'Back')
    expect(getShownWords(tab)).toStrictEqual(RECOVERY_WORDS)
    expect(tab.find('[role="alert"]').exists()).toBe(false)
  })

  it('drops the words from the page when the person cancels the first step', async () => {
    const { tab, create } = await createPage()
    await press(tab, 'Create identity')
    await press(tab, 'Cancel')
    expect(create).not.toHaveBeenCalled()
    for (const word of RECOVERY_WORDS) expect(tab.text()).not.toContain(word)
    expect(findButton(tab, 'Create identity')).toBeDefined()
  })

  it('drops the words from the page when the person presses Escape in the second step', async () => {
    const { tab, create } = await createPage()
    await startSetup(tab)
    await tab.get('[role="dialog"]').trigger('keydown.esc')
    expect(create).not.toHaveBeenCalled()
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(findButton(tab, 'Create identity')).toBeDefined()
  })

  it('asks for new words each time a setup starts', async () => {
    const { tab, generateWords } = await createPage()
    await press(tab, 'Create identity')
    await press(tab, 'Cancel')
    await press(tab, 'Create identity')
    expect(generateWords).toHaveBeenCalledTimes(2)
  })

  it('restores in a dialog with a field for each of the 12 words', async () => {
    const { tab } = await createPage()
    await press(tab, 'Restore')
    expect(getDialog(tab).get('h2').text()).toBe('Restore identity')
    expect(getRestoreInputs(tab).map((input) => input.attributes('aria-label'))).toStrictEqual(
      Array.from({ length: 12 }, (_, index) => `Recovery word ${index + 1}`),
    )
  })

  it('starts a restore with the words typed one to a field when the person restores', async () => {
    const { tab, create, restore } = await createPage()
    await press(tab, 'Restore')
    for (const [index, input] of getRestoreInputs(tab).entries())
      await input.setValue(` ${(RECOVERY_WORDS[index] ?? '').toUpperCase()} `)
    await submitRestore(tab)
    expect(restore).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(create).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
  })

  it('fills the fields after the first when the 12 words are pasted into it', async () => {
    const { tab, restore } = await createPage()
    await press(tab, 'Restore')
    await getRestoreInputs(tab)[0]?.setValue(`  ${RECOVERY_WORDS.join('  ')}\n`)
    expect(
      getRestoreInputs(tab).map((input) => (input.element as HTMLInputElement).value),
    ).toStrictEqual(RECOVERY_WORDS)
    await submitRestore(tab)
    expect(restore).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
  })

  it('says under the fields that the words are not valid and holds no identity when the core refuses them', async () => {
    const { tab, identity } = await createPage(undefined, () =>
      Promise.resolve(asFailure('Some words are not valid.')),
    )
    await press(tab, 'Restore')
    await getRestoreInputs(tab)[0]?.setValue('abandon '.repeat(12))
    await submitRestore(tab)
    const alert = getDialog(tab).get('[role="alert"]')
    expect(alert.text()).toBe('Some words are not valid.')
    const last = getRestoreInputs(tab).at(-1)?.element
    const submit = getDialog(tab).get('button[type="submit"]').element
    if (!last) throw new Error('the dialog has no fields')
    expect(
      last.compareDocumentPosition(alert.element) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      alert.element.compareDocumentPosition(submit) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(identity.value).toBeUndefined()
    expect(tab.text()).toContain(NO_IDENTITY)
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
    await press(tab, 'Restore')
    await getRestoreInputs(tab)[0]?.setValue(RECOVERY_WORDS.join(' '))

    await submitRestore(tab)

    expect(getDialog(tab).get('button[type="submit"]').text()).toBe(words.identityRestoring)
    expect(getDialog(tab).get('button[type="submit"]').attributes('disabled')).toBeDefined()
    expect(identity.value).toBeUndefined()
    finish(asValue(IDENTITY))
    await flushPromises()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('closes the restore dialog on Cancel and drops the words typed', async () => {
    const { tab, restore } = await createPage()
    await press(tab, 'Restore')
    await getRestoreInputs(tab)[0]?.setValue(RECOVERY_WORDS.join(' '))
    await getDialog(tab)
      .findAll('button')
      .find((one) => one.text() === 'Cancel')
      ?.trigger('click')
    await flushPromises()
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(restore).not.toHaveBeenCalled()
    await press(tab, 'Restore')
    expect(
      getRestoreInputs(tab).every((input) => (input.element as HTMLInputElement).value === ''),
    ).toBe(true)
  })

  it('warns that the identity cannot be restored without the words before it deletes', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete identity')
    await nextTick()
    const dialog = document.body.querySelector('[role="alertdialog"]')
    expect(dialog?.textContent).toContain('Without the 12 words this identity cannot be restored.')
    expect(remove).not.toHaveBeenCalled()
  })

  it('keeps the identity when the person cancels the delete', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete identity')
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
    await press(tab, 'Delete identity')
    const confirm = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button'),
    ).find((one) => one.textContent?.trim() === 'Delete')
    confirm?.click()
    await flushPromises()
    expect(remove).toHaveBeenCalledOnce()
    expect(tab.text()).toContain(NO_IDENTITY)
  })
})

const SHEET_ONE = createSheetWords(1)
const SHEET_THREE = createSheetWords(3)

const getSheetField = (tab: Page, number: number) =>
  getDialog(tab).get(`textarea[aria-label="Sheet ${number}"]`)

const getDownload = (tab: Page) =>
  tab
    .findAll('button')
    .find((one) => one.text() === 'Download PDF' && one.attributes('type') === 'button')

const pressDownload = async (tab: Page) => {
  const button = getDownload(tab)
  if (!button) throw new Error('the page has no button "Download PDF"')
  await button.trigger('click')
  await flushPromises()
}

const submitDialog = async (tab: Page) => {
  await getDialog(tab).get('button[type="submit"]').trigger('click')
  await flushPromises()
}

const chooseSheets = async (tab: Page) => {
  const choice = getDialog(tab)
    .findAll('button')
    .find((one) => one.text() === 'Recovery kit sheets')
  if (!choice) throw new Error('the dialog offers no sheets')
  await choice.trigger('click')
  await flushPromises()
}

const keepNewIdentity = async (tab: Page) => {
  await startSetup(tab)
  await fillConfirmations(tab)
  await press(tab, 'Continue')
}

describe('the recovery kit on the page "Identity and licenses"', () => {
  it('is one row of the identity card with a button that downloads a PDF', async () => {
    const { tab } = await createPage(IDENTITY)
    expect(tab.text()).toContain('Recovery kit')
    expect(tab.text()).toContain(
      'Three sheets. Any two restore your account; one alone reveals nothing. Keep them in different places.',
    )
    expect(getDownload(tab)).toBeDefined()
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
  })

  it('saves the kit for the words of the setup without asking again', async () => {
    const { tab, saveKit } = await createPage()
    await keepNewIdentity(tab)
    expect(saveKit).not.toHaveBeenCalled()
    await pressDownload(tab)
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ words: RECOVERY_WORDS })
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(tab.get('[role="status"]').text()).toBe('The recovery kit is saved.')
    for (const word of RECOVERY_WORDS) expect(tab.text()).not.toContain(word)
  })

  it('asks for the words again once the kit of the setup is saved', async () => {
    const { tab, saveKit } = await createPage()
    await keepNewIdentity(tab)
    await pressDownload(tab)
    await pressDownload(tab)
    expect(saveKit).toHaveBeenCalledOnce()
    expect(getDialog(tab).get('h2').text()).toBe('Download recovery kit')
  })

  it('keeps the words of the setup when the person closes the save dialog', async () => {
    const { tab, saveKit } = await createPage(undefined, undefined, { saveKit: asValue(false) })
    await keepNewIdentity(tab)
    await pressDownload(tab)
    await pressDownload(tab)
    expect(saveKit).toHaveBeenCalledTimes(2)
    expect(saveKit).toHaveBeenLastCalledWith({ words: RECOVERY_WORDS })
    expect(tab.find('[role="status"]').exists()).toBe(false)
  })

  it('asks for the 12 words in a dialog and saves the kit for them', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    expect(saveKit).not.toHaveBeenCalled()
    await getRestoreInputs(tab)[0]?.setValue(RECOVERY_WORDS.join(' '))
    await submitDialog(tab)
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ words: RECOVERY_WORDS })
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(tab.get('[role="status"]').text()).toBe('The recovery kit is saved.')
  })

  it('saves the kit for two sheets of an earlier kit', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    await chooseSheets(tab)
    await getSheetField(tab, 1).setValue(SHEET_ONE.join(' '))
    await getSheetField(tab, 2).setValue(SHEET_THREE.join('\n'))
    await submitDialog(tab)
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ sheets: [SHEET_ONE, SHEET_THREE] })
  })

  it('shows a refusal under the fields and keeps the dialog open', async () => {
    const { tab } = await createPage(IDENTITY, undefined, {
      saveKit: asFailure('The words belong to another account.'),
    })
    await pressDownload(tab)
    await getRestoreInputs(tab)[0]?.setValue(RECOVERY_WORDS.join(' '))
    await submitDialog(tab)
    expect(getDialog(tab).get('[role="alert"]').text()).toBe('The words belong to another account.')
    expect(tab.find('[role="status"]').exists()).toBe(false)
  })

  it('asks for one more sheet before it calls the core with none', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    await chooseSheets(tab)
    await submitDialog(tab)
    expect(saveKit).not.toHaveBeenCalled()
    expect(getDialog(tab).get('[role="alert"]').text()).toBe('One more sheet is needed.')
  })

  it('drops the typed words when the person cancels the dialog', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    await getRestoreInputs(tab)[0]?.setValue(RECOVERY_WORDS.join(' '))
    await press(tab, 'Cancel')
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    await pressDownload(tab)
    expect(
      getRestoreInputs(tab).every((input) => (input.element as HTMLInputElement).value === ''),
    ).toBe(true)
    expect(saveKit).not.toHaveBeenCalled()
  })

  it('shows no sheet, no print button and no check of sheets', async () => {
    const { tab } = await createPage()
    await keepNewIdentity(tab)
    await pressDownload(tab)
    expect(tab.findAll('article')).toHaveLength(0)
    expect(findButton(tab, 'Print')).toBeUndefined()
    expect(findButton(tab, 'Check two sheets')).toBeUndefined()
    expect(findButton(tab, 'Skip for now')).toBeUndefined()
  })
})

describe('the restore from sheets on the page "Identity and licenses"', () => {
  it('offers the sheets in the restore dialog and no header action of its own', async () => {
    const { tab } = await createPage()
    expect(findButton(tab, 'Restore from sheets')).toBeUndefined()
    await press(tab, 'Restore')
    await chooseSheets(tab)
    expect(getSheetField(tab, 1).element).toBeInstanceOf(HTMLTextAreaElement)
    expect(getSheetField(tab, 2).element).toBeInstanceOf(HTMLTextAreaElement)
    expect(getRestoreInputs(tab)).toHaveLength(0)
  })

  it('sends the sheets that are typed and shows the identity', async () => {
    const { tab, restoreFromKit, restore } = await createPage()
    await press(tab, 'Restore')
    await chooseSheets(tab)
    await getSheetField(tab, 1).setValue(` ${SHEET_THREE.join('  ')} `)
    await getSheetField(tab, 2).setValue(SHEET_ONE.join(' ').toUpperCase())
    await submitDialog(tab)
    expect(restoreFromKit).toHaveBeenCalledExactlyOnceWith([SHEET_THREE, SHEET_ONE])
    expect(restore).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('sends one sheet when only one is typed and holds no identity after the refusal', async () => {
    const { tab, restoreFromKit, identity } = await createPage(undefined, undefined, {
      restoreFromKit: asFailure('One more sheet is needed.'),
    })
    await press(tab, 'Restore')
    await chooseSheets(tab)
    await getSheetField(tab, 2).setValue(SHEET_ONE.join(' '))
    await submitDialog(tab)
    expect(restoreFromKit).toHaveBeenCalledExactlyOnceWith([SHEET_ONE])
    expect(getDialog(tab).get('[role="alert"]').text()).toBe('One more sheet is needed.')
    expect(identity.value).toBeUndefined()
  })

  it('drops the typed sheets when the person cancels', async () => {
    const { tab, restoreFromKit } = await createPage()
    await press(tab, 'Restore')
    await chooseSheets(tab)
    await getSheetField(tab, 1).setValue(SHEET_ONE.join(' '))
    await getDialog(tab)
      .findAll('button')
      .find((one) => one.text() === 'Cancel')
      ?.trigger('click')
    await flushPromises()
    await press(tab, 'Restore')
    await chooseSheets(tab)
    expect((getSheetField(tab, 1).element as HTMLTextAreaElement).value).toBe('')
    expect(restoreFromKit).not.toHaveBeenCalled()
  })
})
