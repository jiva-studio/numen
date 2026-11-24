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
import {
  createInertDevices,
  createInertKeyFile,
  createInstallation,
  createSettingsState,
} from '@/pages/settings/testing/installation'
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
    identity: {
      identity,
      load: calls.load,
      generateWords: calls.generateWords,
      create: calls.create,
      restore: (proof: KitProof) =>
        'words' in proof ? calls.restore(proof.words) : kitCalls.restoreFromKit(proof.sheets),
      delete: calls.delete,
      saveKit: kitCalls.saveKit,
      devices: createInertDevices(),
      keyFile: createInertKeyFile(),
    },
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

const getProofField = (tab: Page) =>
  getDialog(tab).get('textarea[aria-label="Recovery words or sheets"]')

const getNote = (tab: Page) => getDialog(tab).get('.proof-field__note')

const getSubmit = (tab: Page) => getDialog(tab).get('button[type="submit"]')

const enterProof = async (tab: Page, text: string) => {
  await getProofField(tab).setValue(text)
  await flushPromises()
}

const submitRestore = async (tab: Page) => {
  await getSubmit(tab).trigger('click')
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

  it('shows the join by a QR code in the header with restore and create', async () => {
    const { tab } = await createPage()
    const header = findButton(tab, 'Join account')?.element.parentElement
    expect(header?.contains(findButton(tab, 'Restore')?.element ?? null)).toBe(true)
    expect(header?.contains(findButton(tab, 'Create identity')?.element ?? null)).toBe(true)
  })

  it('opens the join dialog from the header', async () => {
    const { tab } = await createPage()
    await press(tab, 'Join account')
    expect(getDialog(tab).text()).toContain('Join account')
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
    expect(tab.text()).not.toMatch(/\d+ devices?/)
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

  it('restores in a dialog with one field, a line of help and a button that waits', async () => {
    const { tab } = await createPage()
    await press(tab, 'Restore')
    expect(getDialog(tab).get('h2').text()).toBe('Restore identity')
    expect(getDialog(tab).findAll('textarea')).toHaveLength(1)
    expect(getDialog(tab).findAll('input')).toHaveLength(0)
    expect(getDialog(tab).text()).toContain(
      'Scan the QR of two sheets of your recovery kit, or type the 12 words or the words of two sheets.',
    )
    expect(getSubmit(tab).text()).toBe('Restore the account')
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    expect(getNote(tab).text()).toBe('')
  })

  it('counts the words while they are typed and restores when the 12th is in', async () => {
    const { tab, restore } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, RECOVERY_WORDS.slice(0, 5).join(' '))
    expect(getNote(tab).text()).toBe('5 of 12 words')
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    await enterProof(tab, RECOVERY_WORDS.join(' '))
    expect(getNote(tab).text()).toBe('12 words recognized')
    expect(getSubmit(tab).attributes('disabled')).toBeUndefined()
    expect(restore).not.toHaveBeenCalled()
  })

  it('restores from the 12 words in any case and spacing and shows the identity', async () => {
    const { tab, create, restore } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, `  ${RECOVERY_WORDS.join('  ').toUpperCase()}\n`)
    await submitRestore(tab)
    expect(restore).toHaveBeenCalledExactlyOnceWith(RECOVERY_WORDS)
    expect(create).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
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
    await enterProof(tab, RECOVERY_WORDS.join(' '))

    await submitRestore(tab)

    expect(getSubmit(tab).text()).toBe(words.identityRestoring)
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    expect(getProofField(tab).attributes('disabled')).toBeDefined()
    expect(identity.value).toBeUndefined()
    finish(asValue(IDENTITY))
    await flushPromises()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('returns the focus to Restore when the restore dialog is cancelled', async () => {
    const { tab } = await createPage()
    const opener = findButton(tab, 'Restore')
    ;(opener?.element as HTMLElement).focus()
    await press(tab, 'Restore')
    expect(document.activeElement).not.toBe(opener?.element)
    await press(tab, 'Cancel')
    expect(document.activeElement).toBe(opener?.element)
  })

  it('refuses 12 words followed by an extra one and does not call the core', async () => {
    const { tab, restore } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, `${RECOVERY_WORDS.join(' ')} extra`)
    expect(getNote(tab).text()).toContain('13 words')
    expect(getNote(tab).attributes('role')).toBe('alert')
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    await submitRestore(tab)
    expect(restore).not.toHaveBeenCalled()
  })

  it('says under the field that the words are not valid when the core refuses them', async () => {
    const { tab, identity } = await createPage(undefined, () =>
      Promise.resolve(asFailure('Some words are not valid.')),
    )
    await press(tab, 'Restore')
    await enterProof(tab, 'abandon '.repeat(12))
    await submitRestore(tab)
    const alert = getDialog(tab).get('[role="alert"]')
    expect(alert.text()).toBe('Some words are not valid.')
    expect(
      getProofField(tab).element.compareDocumentPosition(alert.element) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      alert.element.compareDocumentPosition(getSubmit(tab).element) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(identity.value).toBeUndefined()
    expect(tab.text()).toContain(NO_IDENTITY)
    await enterProof(tab, 'abandon '.repeat(11))
    expect(getNote(tab).text()).toBe('11 of 12 words')
  })

  it('closes the restore dialog on Cancel and drops what was entered', async () => {
    const { tab, restore } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, RECOVERY_WORDS.join(' '))
    await press(tab, 'Cancel')
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(restore).not.toHaveBeenCalled()
    await press(tab, 'Restore')
    expect((getProofField(tab).element as HTMLTextAreaElement).value).toBe('')
  })

  it('warns that the identity cannot be restored without the words before it deletes', async () => {
    const { tab, delete: remove } = await createPage(IDENTITY)
    await press(tab, 'Delete identity')
    await nextTick()
    const dialog = document.body.querySelector('[role="alertdialog"]')
    expect(dialog?.textContent).toContain(
      'Without the 12 words or two sheets of the recovery kit this identity cannot be restored.',
    )
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

const readToasts = (): string[] =>
  [...document.body.querySelectorAll('[role="status"]')]
    .map((one) => one.textContent?.trim() ?? '')
    .filter((text) => text !== '')

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
      'Keep each sheet in a different place. Any two of three restore your account; one sheet alone is useless.',
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
    expect(readToasts()).toStrictEqual(['Recovery kit saved'])
    expect(tab.text()).not.toContain('Recovery kit saved')
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
    expect(readToasts()).toStrictEqual([])
  })

  it('asks for the 12 words in a dialog and saves the kit for them', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    expect(saveKit).not.toHaveBeenCalled()
    await enterProof(tab, RECOVERY_WORDS.join(' '))
    await submitRestore(tab)
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ words: RECOVERY_WORDS })
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    expect(readToasts()).toStrictEqual(['Recovery kit saved'])
  })

  it('saves the kit for two sheets of an earlier kit', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    await enterProof(tab, `${SHEET_ONE.join(' ')}\n${SHEET_THREE.join('  ')}`)
    await submitRestore(tab)
    expect(saveKit).toHaveBeenCalledExactlyOnceWith({ sheets: [SHEET_ONE, SHEET_THREE] })
  })

  it('shows a refusal under the fields and keeps the dialog open', async () => {
    const { tab } = await createPage(IDENTITY, undefined, {
      saveKit: asFailure('The words belong to another account.'),
    })
    await pressDownload(tab)
    await enterProof(tab, RECOVERY_WORDS.join(' '))
    await submitRestore(tab)
    expect(getDialog(tab).get('[role="alert"]').text()).toBe('The words belong to another account.')
    expect(readToasts()).toStrictEqual([])
  })

  it('waits for a whole proof before it calls the core', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    await enterProof(tab, SHEET_ONE.join(' '))
    expect(getNote(tab).text()).toBe('Sheet 1 recognized, one more sheet needed')
    await submitRestore(tab)
    expect(saveKit).not.toHaveBeenCalled()
  })

  it('drops the typed words when the person cancels the dialog', async () => {
    const { tab, saveKit } = await createPage(IDENTITY)
    await pressDownload(tab)
    await enterProof(tab, RECOVERY_WORDS.join(' '))
    await press(tab, 'Cancel')
    expect(tab.find('[role="dialog"]').exists()).toBe(false)
    await pressDownload(tab)
    expect((getProofField(tab).element as HTMLTextAreaElement).value).toBe('')
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
  it('recognizes two sheets, one on each line, and sends them', async () => {
    const { tab, restoreFromKit, restore } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, SHEET_THREE.join(' '))
    expect(getNote(tab).text()).toBe('Sheet 1 recognized, one more sheet needed')
    await enterProof(tab, ` ${SHEET_THREE.join('  ')} \n${SHEET_ONE.join(' ').toUpperCase()}`)
    expect(getNote(tab).text()).toBe('2 of 3 sheets recognized')
    await submitRestore(tab)
    expect(restoreFromKit).toHaveBeenCalledExactlyOnceWith([SHEET_THREE, SHEET_ONE])
    expect(restore).not.toHaveBeenCalled()
    expect(tab.text()).toContain(IDENTITY.accountFingerprint)
  })

  it('does not send one sheet and asks for one more', async () => {
    const { tab, restoreFromKit, identity } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, SHEET_ONE.join(' '))
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    await submitRestore(tab)
    expect(restoreFromKit).not.toHaveBeenCalled()
    expect(identity.value).toBeUndefined()
  })

  it('refuses a repeated sheet and names it', async () => {
    const { tab, restoreFromKit } = await createPage()
    await press(tab, 'Restore')
    await enterProof(tab, `${SHEET_ONE.join(' ')}\n${SHEET_ONE.join(' ')}`)
    expect(getNote(tab).text()).toBe('Sheet 2 is the same as sheet 1. Enter a different sheet.')
    expect(getNote(tab).attributes('role')).toBe('alert')
    expect(getSubmit(tab).attributes('disabled')).toBeDefined()
    expect(restoreFromKit).not.toHaveBeenCalled()
  })

  it('shows the refusal of the core for sheets of another account under the field', async () => {
    const { tab, identity } = await createPage(undefined, undefined, {
      restoreFromKit: asFailure('The sheets belong to another account.'),
    })
    await press(tab, 'Restore')
    await enterProof(tab, `${SHEET_ONE.join(' ')}\n${SHEET_THREE.join(' ')}`)
    await submitRestore(tab)
    expect(getNote(tab).text()).toBe('The sheets belong to another account.')
    expect(identity.value).toBeUndefined()
  })
})
