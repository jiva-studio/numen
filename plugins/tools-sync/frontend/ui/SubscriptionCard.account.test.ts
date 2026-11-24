import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { asValue, type PluginContext } from '@numen/plugin-sdk'
import type { AccountClient } from '../api/account'
import type { SaveFile } from '../api/saveFile'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const ACCOUNT = '0123456789abcdef0123456789abcdef'
const DELETE_AT = '2026-11-30T12:00:00Z'
const EXPORT_DOCUMENT = '{"account":"0123456789abcdef0123456789abcdef","vaults":[]}'

const ACTIVE: SubscriptionResponse = {
  hasIdentity: true,
  phase: 'active',
  periodEndAt: '2026-12-23T12:00:00Z',
  writesEndAt: '',
  deleteAt: '',
  plan: 'plus',
  quotaBytes: 100 * 2 ** 30,
  usedBytes: 2 ** 30,
}

type AccountStateResponse = Awaited<ReturnType<AccountClient['getAccountState']>>

const createState = (patch: Partial<AccountStateResponse> = {}): AccountStateResponse => ({
  hasIdentity: true,
  account: ACCOUNT,
  isDeleting: false,
  deleteAt: '',
  ...patch,
})

const createAccountClient = (state: AccountStateResponse) => {
  const current = { ...state }
  return {
    getAccountState: vi.fn<AccountClient['getAccountState']>(() => Promise.resolve({ ...current })),
    deleteCloudData: vi.fn<AccountClient['deleteCloudData']>(() => {
      current.isDeleting = true
      current.deleteAt = DELETE_AT
      return Promise.resolve(DELETE_AT)
    }),
    restoreAccount: vi.fn<AccountClient['restoreAccount']>(() => {
      current.isDeleting = false
      current.deleteAt = ''
      return Promise.resolve()
    }),
    exportAccountData: vi.fn<AccountClient['exportAccountData']>(() =>
      Promise.resolve(EXPORT_DOCUMENT),
    ),
  }
}

const mounted: VueWrapper[] = []

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
})

const render = async (state: AccountStateResponse, subscription = ACTIVE) => {
  const accountClient = createAccountClient(state)
  const client = {
    getSubscription: vi.fn<SubscriptionClient['getSubscription']>(() =>
      Promise.resolve(subscription),
    ),
    listPlans: vi.fn<SubscriptionClient['listPlans']>(() => Promise.resolve([])),
    changePlan: vi.fn<SubscriptionClient['changePlan']>(() => Promise.resolve(subscription)),
    createCheckout: vi.fn<SubscriptionClient['createCheckout']>(() => Promise.resolve('')),
    createPortalSession: vi.fn<SubscriptionClient['createPortalSession']>(() =>
      Promise.resolve(''),
    ),
  }
  const saveFile = vi.fn<SaveFile>(() => Promise.resolve('/home/me/Numen Sync account data.json'))
  const openExternal = vi.fn<PluginContext['identity']['openExternal']>(() =>
    Promise.resolve(asValue(null)),
  )
  const wrapper = mount(SubscriptionCard, {
    props: {
      client,
      accountClient,
      saveFile,
      openExternal,
      now: () => new Date('2026-11-23T12:00:00Z'),
    },
    attachTo: document.body,
  })
  mounted.push(wrapper)
  await flushPromises()
  return { wrapper, client, accountClient, saveFile }
}

const findButton = (text: string) =>
  Array.from(document.body.querySelectorAll('button')).find(
    (one) => (one.getAttribute('aria-label') ?? one.textContent?.trim()) === text,
  )

const press = async (text: string) => {
  findButton(text)?.click()
  await flushPromises()
}

const pressMenu = async (text: string) => {
  await press('More actions')
  await press(text)
}

const pressInDialog = async (text: string) => {
  Array.from(document.body.querySelectorAll('[role="alertdialog"] button'))
    .find((one) => one.textContent?.trim() === text)
    ?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
  await flushPromises()
}

const type = async (text: string) => {
  const input = document.body.querySelector<HTMLInputElement>('.subscription__input')
  if (input === null) throw new Error('the dialog has no field')
  input.value = text
  input.dispatchEvent(new Event('input'))
  await flushPromises()
}

const getDialogText = (): string =>
  document.body.querySelector('[role="alertdialog"]')?.textContent ?? ''

describe('SubscriptionCard account actions', () => {
  it('offers to export the account data and to delete the cloud data in the menu of the header', async () => {
    await render(createState())
    expect(findButton('Export account data')).toBeUndefined()
    await press('More actions')

    expect(findButton('Export account data')).toBeDefined()
    expect(findButton('Delete cloud data')).toBeDefined()
    expect(findButton('Delete identity')).toBeUndefined()
  })

  it('shows no account action on a card that has no account client', async () => {
    const wrapper = mount(SubscriptionCard, {
      props: {
        client: {
          getSubscription: () => Promise.resolve(ACTIVE),
          listPlans: () => Promise.resolve([]),
          changePlan: () => Promise.resolve(ACTIVE),
          createCheckout: () => Promise.resolve(''),
          createPortalSession: () => Promise.resolve(''),
        },
        openExternal: () => Promise.resolve(asValue(null)),
      },
      attachTo: document.body,
    })
    mounted.push(wrapper)
    await flushPromises()

    expect(findButton('More actions')).toBeUndefined()
  })

  it('asks what is deleted, the seven days, that notes stay and that the subscription is cancelled', async () => {
    await render(createState())

    await pressMenu('Delete cloud data')

    const text = getDialogText()
    expect(text).toContain('Delete your cloud data?')
    expect(text).toContain('7 days')
    expect(text).toContain('Your notes stay on your devices')
    expect(text).toContain('Your subscription is cancelled now')
    expect(text).toContain(ACCOUNT)
  })

  it('does not mention a subscription to cancel when there is none', async () => {
    await render(createState(), { ...ACTIVE, phase: 'none', plan: '', quotaBytes: 0 })

    await pressMenu('Delete cloud data')

    expect(getDialogText()).not.toContain('Your subscription is cancelled now')
  })

  it('deletes nothing and shows under the field that the typed text is not the account name', async () => {
    const { accountClient } = await render(createState())
    await pressMenu('Delete cloud data')

    await type('somebody else')
    await pressInDialog('Delete cloud data')

    expect(accountClient.deleteCloudData).not.toHaveBeenCalled()
    expect(document.body.querySelector('[role="alertdialog"] [role="alert"]')?.textContent).toBe(
      'This is not the account fingerprint.',
    )
  })

  it('deletes the cloud data once the account name is typed and shows the deleting state with the date', async () => {
    const { wrapper, accountClient } = await render(createState())
    await pressMenu('Delete cloud data')

    await type(ACCOUNT)
    await pressInDialog('Delete cloud data')

    expect(accountClient.deleteCloudData).toHaveBeenCalledOnce()
    expect(document.body.querySelector('[role="alertdialog"]')).toBeNull()
    expect(wrapper.get('.subscription__badge').text()).toBe('Deleting')
    expect(wrapper.text()).toContain('deleted on November 30, 2026')
    expect(wrapper.text()).toContain('Restore account')
  })

  it('does not delete when the person cancels the question', async () => {
    const { accountClient } = await render(createState())
    await pressMenu('Delete cloud data')

    await pressInDialog('Cancel')

    expect(accountClient.deleteCloudData).not.toHaveBeenCalled()
    expect(document.body.querySelector('[role="alertdialog"]')).toBeNull()
  })

  it('shows the deleting state of an account that is already deleting and stops reading the subscription', async () => {
    const { wrapper, client } = await render(createState({ isDeleting: true, deleteAt: DELETE_AT }))

    expect(wrapper.text()).toContain('Restore account')
    expect(wrapper.text()).not.toContain('Delete cloud data')
    expect(client.getSubscription).not.toHaveBeenCalled()
  })

  it('restores the account and brings the card back', async () => {
    const { wrapper, accountClient } = await render(
      createState({ isDeleting: true, deleteAt: DELETE_AT }),
    )

    await press('Restore account')

    expect(accountClient.restoreAccount).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toContain('Restore account')
  })

  it('saves the document the server sent under the name of the file', async () => {
    const { saveFile } = await render(createState())

    await pressMenu('Export account data')

    expect(saveFile).toHaveBeenCalledWith('Numen Sync account data.json', EXPORT_DOCUMENT)
  })

  it('says where the file was saved and that it is not encrypted', async () => {
    const { wrapper } = await render(createState())

    await pressMenu('Export account data')

    const text = wrapper.get('[role="status"]').text()
    expect(text).toContain('Saved to /home/me/Numen Sync account data.json')
    expect(text).toContain('This file is not encrypted. Delete it after use.')
  })

  it('says nothing when the person closes the save dialog', async () => {
    const { wrapper, saveFile } = await render(createState())
    saveFile.mockResolvedValueOnce(undefined)

    await pressMenu('Export account data')

    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows that the export is under way until the file is saved', async () => {
    const { wrapper, saveFile } = await render(createState())
    const saving = Promise.withResolvers<string | undefined>()
    saveFile.mockReturnValueOnce(saving.promise)

    await pressMenu('Export account data')
    expect(wrapper.get('[role="status"]').text()).toBe('Exporting account data')

    saving.resolve('/home/me/a.json')
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toContain('Saved to /home/me/a.json')
  })

  it('shows the reason under the card when the file cannot be saved', async () => {
    const { wrapper, saveFile } = await render(createState())
    saveFile.mockRejectedValueOnce(new Error('Could not save the file: the disk is full'))

    await pressMenu('Export account data')

    expect(wrapper.get('[role="alert"]').text()).toBe('Could not save the file: the disk is full')
  })

  it('shows the reason under the card when the export fails', async () => {
    const { wrapper, accountClient } = await render(createState())
    accountClient.exportAccountData.mockRejectedValueOnce(new Error('Could not reach the server'))

    await pressMenu('Export account data')

    expect(wrapper.get('[role="alert"]').text()).toBe('Could not reach the server')
  })
})
