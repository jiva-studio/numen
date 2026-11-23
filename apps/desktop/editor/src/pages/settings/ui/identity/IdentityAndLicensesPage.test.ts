/**
 * The page "Identity and licenses" inside the settings tab: a person sets up an identity in a dialog
 * of two steps (twelve words, then three of them to check), restores it from words and deletes it from this device. The core is
 * replaced by the calls of the installation; the page, its model and the settings tab are real.
 *
 * Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref, type Ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { asFailure, asValue, type Result } from '@numen/result'
import type { IdentityInfo, RecoveryKit } from '@/entities/identity'
import type { LicenseInfo } from '@/entities/license'
import { createInstallation, createSettingsState } from '@/pages/settings/testing/installation'
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
  verifiedAt: '',
}

const createSheetWords = (sheet: number): string[] =>
  Array.from({ length: 33 }, (_, index) => `s${sheet}w${index + 1}`)

const KIT: RecoveryKit = {
  id: '5C21',
  accountFingerprint: IDENTITY.accountFingerprint,
  sheets: [createSheetWords(1), createSheetWords(2), createSheetWords(3)],
}

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
    createKit?: Result<RecoveryKit, string>
    checkKit?: Result<string, string>
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
    delete: vi.fn(() => {
      identity.value = undefined
      return Promise.resolve(asValue<void>(undefined))
    }),
  }
  const kitCalls = {
    createKit: vi.fn((_proof: KitProof) => Promise.resolve(kitAnswers.createKit ?? asValue(KIT))),
    checkKit: vi.fn((_sheets: Sheets) =>
      Promise.resolve(kitAnswers.checkKit ?? asValue(IDENTITY.accountFingerprint)),
    ),
    restoreFromKit: vi.fn((_sheets: Sheets) => {
      const answer = kitAnswers.restoreFromKit ?? asValue(IDENTITY)
      if (answer.ok) identity.value = answer.value
      return Promise.resolve(answer)
    }),
  }
  const printer = { print: vi.fn() }
  const installation = createInstallation({
    licenses: {
      list: ref([SYNC]),
      load: vi.fn(() => Promise.resolve()),
      watch: vi.fn(() => Promise.resolve()),
      activate: vi.fn(() => Promise.resolve(asFailure('Not activated.'))),
      deactivate: vi.fn(() => Promise.resolve(asFailure('Not deactivated.'))),
    },
    identity: { identity, ...calls, ...kitCalls },
    printer,
  })
  const tab = mount(SettingsTab, {
    props: { state: createSettingsState(installation, { initialSearchQuery: ref('') }) },
    attachTo: document.body,
  })
  mounted.push(tab)
  await tab.get('button[aria-label="Identity and licenses"]').trigger('click')
  await flushPromises()
  return { tab, identity, printer, ...calls, ...kitCalls }
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
    expect(tab.text()).toContain('Not created')
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
    expect(tab.get('[role="alert"]').text()).toBe('Some words are not valid.')
    expect(tab.text()).toContain('Not created')
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

  it('sends the words that are typed to create when the person restores', async () => {
    const { tab, create } = await createPage()
    await press(tab, 'Restore')
    await tab
      .get('textarea[aria-label="Your 12 recovery words"]')
      .setValue(`  ${RECOVERY_WORDS.join('  ')}\n`)
    await press(tab, 'Restore')
    expect(create).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('says the words are not valid and holds no identity when the core refuses the words', async () => {
    const { tab, identity } = await createPage(undefined, () =>
      Promise.resolve(asFailure('Some words are not valid.')),
    )
    await press(tab, 'Restore')
    await tab.get('textarea[aria-label="Your 12 recovery words"]').setValue('abandon '.repeat(12))
    await press(tab, 'Restore')
    expect(tab.get('[role="alert"]').text()).toBe('Some words are not valid.')
    expect(identity.value).toBeUndefined()
    expect(tab.text()).toContain('Not created')
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
    expect(tab.text()).toContain('Not created')
  })
})

const getSheets = (tab: Page) =>
  tab.findAll('article').map((sheet) => ({
    label: sheet.attributes('aria-label'),
    text: sheet.text(),
    words: sheet.findAll('ol li').map((word) => word.text()),
  }))

const getSheetField = (tab: Page, number: number) =>
  tab.get(`textarea[aria-label="Sheet ${number}"]`)

const getTypedWords = (tab: Page): string =>
  tab.get<HTMLTextAreaElement>('textarea[aria-label="Recovery words"]').element.value

const getHeadings = (tab: Page): string[] =>
  tab.findAll('h1, h2, h3, h4, [role="heading"]').map((one) => one.text())

const keepNewIdentity = async (tab: Page) => {
  await startSetup(tab)
  await fillConfirmations(tab)
  await press(tab, 'Continue')
}

describe('the recovery kit on the page "Identity and licenses"', () => {
  it('makes the kit with the words of the setup when the identity is kept', async () => {
    const { tab, createKit } = await createPage()
    await keepNewIdentity(tab)
    expect(createKit).toHaveBeenCalledExactlyOnceWith({ words: RECOVERY_WORDS })
    expect(getHeadings(tab)).toContain('Recovery kit')
    expect(tab.text()).toContain('Print three sheets. Any two restore your account.')
    expect(findButton(tab, 'Print')).toBeDefined()
    expect(findButton(tab, 'Check two sheets')).toBeDefined()
    expect(findButton(tab, 'Skip for now')).toBeDefined()
  })

  it('shows three sheets of 33 numbered words', async () => {
    const { tab } = await createPage()
    await keepNewIdentity(tab)
    const sheets = getSheets(tab)
    expect(sheets.map((sheet) => sheet.label)).toStrictEqual([
      'Sheet 1 of 3',
      'Sheet 2 of 3',
      'Sheet 3 of 3',
    ])
    expect(sheets.map((sheet) => sheet.words)).toStrictEqual(KIT.sheets)
    expect(sheets.every((sheet) => sheet.words.length === 33)).toBe(true)
    expect(tab.findAll('article ol')).toHaveLength(3)
  })

  it('writes the kit, the account and the instructions on each sheet', async () => {
    const { tab } = await createPage()
    await keepNewIdentity(tab)
    for (const sheet of getSheets(tab)) {
      expect(sheet.text).toContain('Numen recovery kit')
      expect(sheet.text).toContain('Kit 5C21')
      expect(sheet.text).toContain('Account')
      expect(sheet.text).toContain(IDENTITY.accountFingerprint)
      expect(sheet.text).toContain(
        'Any 2 of the 3 sheets restore your account. One sheet alone shows nothing. Keep each sheet in a different safe place.',
      )
      expect(sheet.text).toContain(
        'Numen cannot restore your account without the words or two sheets.',
      )
    }
  })

  it('sends the kit to the printer once and keeps the sheets', async () => {
    const { tab, printer, createKit } = await createPage()
    await keepNewIdentity(tab)
    await press(tab, 'Print')
    expect(printer.print).toHaveBeenCalledOnce()
    expect(getSheets(tab)).toHaveLength(3)
    expect(createKit).toHaveBeenCalledOnce()
  })

  it('does not print before the person asks', async () => {
    const { tab, printer } = await createPage()
    await keepNewIdentity(tab)
    expect(printer.print).not.toHaveBeenCalled()
  })

  it('drops the kit and the words when the person skips it', async () => {
    const { tab, createKit } = await createPage()
    await keepNewIdentity(tab)
    await press(tab, 'Skip for now')
    expect(getSheets(tab)).toHaveLength(0)
    expect(getHeadings(tab)).not.toContain('Recovery kit')
    for (const word of [...RECOVERY_WORDS, ...KIT.sheets.flat()]) {
      expect(tab.text()).not.toContain(word)
    }
    expect(findButton(tab, 'Print recovery kit')).toBeDefined()
    expect(findButton(tab, 'Check two sheets')).toBeDefined()
    await press(tab, 'Print recovery kit')
    expect(createKit).toHaveBeenCalledOnce()
    expect(getSheets(tab)).toHaveLength(0)
    expect(getTypedWords(tab)).toBe('')
  })

  it('makes no kit when the setup is cancelled', async () => {
    const { tab, createKit } = await createPage()
    await press(tab, 'Create identity')
    await press(tab, 'Cancel')
    expect(createKit).not.toHaveBeenCalled()
  })

  it('asks for the words or two sheets before it makes a new kit', async () => {
    const { tab, createKit } = await createPage(IDENTITY)
    expect(findButton(tab, 'Print recovery kit')).toBeDefined()
    expect(findButton(tab, 'Check two sheets')).toBeDefined()
    await press(tab, 'Print recovery kit')
    expect(createKit).not.toHaveBeenCalled()
    expect(tab.find('textarea[aria-label="Recovery words"]').exists()).toBe(true)
    expect(tab.find('textarea[aria-label="Sheet 1"]').exists()).toBe(true)
    expect(tab.find('textarea[aria-label="Sheet 2"]').exists()).toBe(true)
    expect(tab.text()).toContain(
      'Enter the 12 words or two sheets of an earlier kit to print a new kit.',
    )
    expect(tab.text()).toContain('A new kit does not cancel an older kit.')
  })

  it('makes a kit for the words that are typed', async () => {
    const { tab, createKit } = await createPage(IDENTITY)
    await press(tab, 'Print recovery kit')
    await tab
      .get('textarea[aria-label="Recovery words"]')
      .setValue(`  ${RECOVERY_WORDS.join('  ').toUpperCase()}\n`)
    await press(tab, 'Make kit')
    expect(createKit).toHaveBeenCalledExactlyOnceWith({ words: RECOVERY_WORDS })
    expect(getSheets(tab).map((sheet) => sheet.words)).toStrictEqual(KIT.sheets)
  })

  it('makes a kit for two sheets of an earlier kit', async () => {
    const { tab, createKit } = await createPage(IDENTITY)
    await press(tab, 'Print recovery kit')
    await getSheetField(tab, 1).setValue(KIT.sheets[0]?.join(' ') ?? '')
    await getSheetField(tab, 2).setValue(KIT.sheets[2]?.join('\n') ?? '')
    await press(tab, 'Make kit')
    expect(createKit).toHaveBeenCalledExactlyOnceWith({ sheets: [KIT.sheets[0], KIT.sheets[2]] })
    expect(getSheets(tab)).toHaveLength(3)
  })

  it('shows the refusal and no sheet when the proof is refused', async () => {
    const { tab } = await createPage(IDENTITY, undefined, {
      createKit: asFailure('The words belong to another account.'),
    })
    await press(tab, 'Print recovery kit')
    await tab.get('textarea[aria-label="Recovery words"]').setValue(RECOVERY_WORDS.join(' '))
    await press(tab, 'Make kit')
    expect(tab.get('[role="alert"]').text()).toBe('The words belong to another account.')
    expect(getSheets(tab)).toHaveLength(0)
  })

  it('drops the typed words when the person cancels the proof', async () => {
    const { tab } = await createPage(IDENTITY)
    await press(tab, 'Print recovery kit')
    await tab.get('textarea[aria-label="Recovery words"]').setValue(RECOVERY_WORDS.join(' '))
    await press(tab, 'Cancel')
    expect(tab.find('textarea[aria-label="Recovery words"]').exists()).toBe(false)
    await press(tab, 'Print recovery kit')
    expect(getTypedWords(tab)).toBe('')
  })

  it('shows two fields for the sheets and confirms that the sheets fit the account', async () => {
    const { tab, checkKit } = await createPage(IDENTITY)
    await press(tab, 'Check two sheets')
    expect(checkKit).not.toHaveBeenCalled()
    await getSheetField(tab, 1).setValue(`  ${KIT.sheets[0]?.join(' ').toUpperCase()}\n`)
    await getSheetField(tab, 2).setValue(KIT.sheets[2]?.join('\n') ?? '')
    await press(tab, 'Check')
    expect(checkKit).toHaveBeenCalledExactlyOnceWith([KIT.sheets[0], KIT.sheets[2]])
    const statuses = tab.findAll('[role="status"]').map((one) => one.text())
    expect(statuses).toContain('The sheets fit this account.')
  })

  it('shows a refusal of the check as an alert and no confirmation', async () => {
    const { tab } = await createPage(IDENTITY, undefined, {
      checkKit: asFailure('Some sheets are not valid.'),
    })
    await press(tab, 'Check two sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[0]?.join(' ') ?? '')
    await getSheetField(tab, 2).setValue(KIT.sheets[1]?.join(' ') ?? '')
    await press(tab, 'Check')
    expect(tab.get('[role="alert"]').text()).toBe('Some sheets are not valid.')
    expect(tab.text()).not.toContain('The sheets fit this account.')
  })

  it('sends only the sheets that are typed', async () => {
    const { tab, checkKit } = await createPage(IDENTITY)
    await press(tab, 'Check two sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[1]?.join(' ') ?? '')
    await press(tab, 'Check')
    expect(checkKit).toHaveBeenCalledExactlyOnceWith([KIT.sheets[1]])
  })

  it('checks the sheets from the kit step without a new kit', async () => {
    const { tab, createKit, checkKit } = await createPage()
    await keepNewIdentity(tab)
    await press(tab, 'Check two sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[0]?.join(' ') ?? '')
    await getSheetField(tab, 2).setValue(KIT.sheets[1]?.join(' ') ?? '')
    await press(tab, 'Check')
    expect(checkKit).toHaveBeenCalledOnce()
    expect(createKit).toHaveBeenCalledOnce()
  })
})

describe('the restore from sheets on the page "Identity and licenses"', () => {
  it('offers the restore from sheets next to the restore from words', async () => {
    const { tab } = await createPage()
    expect(findButton(tab, 'Restore from sheets')).toBeDefined()
    expect(findButton(tab, 'Restore')).toBeDefined()
  })

  it('offers no restore from sheets when the device has an identity', async () => {
    const { tab } = await createPage(IDENTITY)
    expect(findButton(tab, 'Restore from sheets')).toBeUndefined()
  })

  it('sends the sheets that are typed and shows the identity', async () => {
    const { tab, restoreFromKit } = await createPage()
    await press(tab, 'Restore from sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[1]?.join(' ') ?? '')
    await getSheetField(tab, 2).setValue(`${KIT.sheets[2]?.join('\n')}\n`)
    await press(tab, 'Restore')
    expect(restoreFromKit).toHaveBeenCalledExactlyOnceWith([KIT.sheets[1], KIT.sheets[2]])
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.find('textarea[aria-label="Sheet 1"]').exists()).toBe(false)
    for (const word of KIT.sheets.flat()) expect(tab.text()).not.toContain(word)
  })

  it('sends one sheet when only one is typed and holds no identity after the refusal', async () => {
    const { tab, restoreFromKit, identity } = await createPage(undefined, undefined, {
      restoreFromKit: asFailure('One more sheet is needed.'),
    })
    await press(tab, 'Restore from sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[0]?.join(' ') ?? '')
    await press(tab, 'Restore')
    expect(restoreFromKit).toHaveBeenCalledExactlyOnceWith([KIT.sheets[0]])
    expect(tab.get('[role="alert"]').text()).toBe('One more sheet is needed.')
    expect(identity.value).toBeUndefined()
    expect(tab.text()).toContain('Not created')
  })

  it('drops the typed sheets when the person cancels', async () => {
    const { tab, restoreFromKit } = await createPage()
    await press(tab, 'Restore from sheets')
    await getSheetField(tab, 1).setValue(KIT.sheets[0]?.join(' ') ?? '')
    await press(tab, 'Cancel')
    expect(restoreFromKit).not.toHaveBeenCalled()
    await press(tab, 'Restore from sheets')
    expect(getSheetField(tab, 1).element).toHaveProperty('value', '')
  })
})
