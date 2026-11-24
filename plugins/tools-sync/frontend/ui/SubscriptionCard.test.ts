import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { asValue, type PluginContext } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const NOW = new Date('2026-11-23T12:00:00Z')
const CHECKOUT_URL = 'https://pay.example/checkout/abc'
const PORTAL_URL = 'https://portal.example/session/1'

const GIB = 2 ** 30
const PLANS = [
  { id: 'basic', name: 'Basic', quotaBytes: 10 * GIB, priceCents: 500, currency: 'USD' },
  { id: 'plus', name: 'Plus', quotaBytes: 100 * GIB, priceCents: 1500, currency: 'USD' },
  { id: 'pro', name: 'Pro', quotaBytes: 250 * GIB, priceCents: 3000, currency: 'USD' },
] as const
const quotaOf = (plan: string): number => PLANS.find((one) => one.id === plan)?.quotaBytes ?? 0

const createResponse = (patch: Partial<SubscriptionResponse> = {}): SubscriptionResponse => ({
  hasIdentity: true,
  phase: 'none',
  periodEndAt: '',
  writesEndAt: '',
  deleteAt: '',
  plan: '',
  quotaBytes: 0,
  usedBytes: 0,
  ...patch,
})

const ACTIVE_ON_PLUS = createResponse({
  phase: 'active',
  periodEndAt: '2026-12-23T12:00:00Z',
  plan: 'plus',
  quotaBytes: 100 * GIB,
  usedBytes: 2 * GIB,
})

const createClient = (answer: () => Promise<SubscriptionResponse>) => ({
  getSubscription: vi.fn<SubscriptionClient['getSubscription']>(answer),
  listPlans: vi.fn<SubscriptionClient['listPlans']>(() => Promise.resolve(PLANS)),
  changePlan: vi.fn<SubscriptionClient['changePlan']>((plan) =>
    Promise.resolve(createResponse({ ...ACTIVE_ON_PLUS, plan, quotaBytes: quotaOf(plan) })),
  ),
  createCheckout: vi.fn<SubscriptionClient['createCheckout']>(() => Promise.resolve(CHECKOUT_URL)),
  createPortalSession: vi.fn<SubscriptionClient['createPortalSession']>(() =>
    Promise.resolve(PORTAL_URL),
  ),
})

const mounted: VueWrapper[] = []

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
})

const mountCard = (props: InstanceType<typeof SubscriptionCard>['$props']) => {
  const wrapper = mount(SubscriptionCard, { props, attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}

const render = async (response: SubscriptionResponse) => {
  const client = createClient(() => Promise.resolve(response))
  const openExternal = vi.fn<PluginContext['identity']['openExternal']>(() =>
    Promise.resolve(asValue(null)),
  )
  const wrapper = mountCard({ client, openExternal, now: () => NOW })
  await flushPromises()
  return { wrapper, client, openExternal }
}

const findChoose = (wrapper: VueWrapper, name?: string) =>
  wrapper
    .findAll('button')
    .find((one) => one.attributes('aria-label')?.startsWith(`Choose ${name ?? ''}`) === true)

const getChooseNames = (wrapper: VueWrapper): string[] =>
  wrapper
    .findAll('button')
    .map((one) => one.attributes('aria-label') ?? '')
    .filter((name) => name.startsWith('Choose '))

const getRowNames = (wrapper: VueWrapper): string[] =>
  wrapper.findAll('.subscription__row .subscription__name').map((one) => one.text())

const getRow = (wrapper: VueWrapper, name: string): string =>
  wrapper
    .findAll('.subscription__row')
    .find((row) => row.find('.subscription__name').text() === name)
    ?.text() ?? ''

const getContent = (wrapper: VueWrapper) => wrapper.get('.subscription__content')

const findCancel = (wrapper: VueWrapper) =>
  wrapper.findAll('button').find((one) => one.text() === 'Cancel')

const toggleMenu = async (wrapper: VueWrapper) => {
  const trigger = wrapper.find('button[aria-label="More actions"]')
  if (trigger.exists()) await trigger.trigger('click')
  await flushPromises()
}

const findMenuItem = (text: string) =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')).find(
    (one) => one.textContent?.trim() === text,
  )

const hasMenuItem = async (wrapper: VueWrapper, text: string) => {
  await toggleMenu(wrapper)
  const isThere = findMenuItem(text) !== undefined
  await toggleMenu(wrapper)
  return isThere
}

const getPlanItems = () =>
  Array.from(document.body.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]'))

const getPlanName = (item: Element): string => {
  const name = Array.from(item.querySelector('.menu__text')?.childNodes ?? []).find(
    (node) => node.textContent?.trim() !== '',
  )
  return name?.textContent?.trim() ?? ''
}

const getPlanChoices = (): string[] => getPlanItems().map(getPlanName)

const openPlans = async (wrapper: VueWrapper) => {
  await toggleMenu(wrapper)
  findMenuItem('Change plan')?.click()
  await flushPromises()
}

const choosePlan = async (wrapper: VueWrapper, name: string) => {
  await openPlans(wrapper)
  getPlanItems()
    .find((one) => getPlanName(one) === name)
    ?.click()
  await flushPromises()
}

const findSwitchDialog = () => document.body.querySelector<HTMLElement>('[role="alertdialog"]')

const getSwitchFacts = (): string[] =>
  Array.from(findSwitchDialog()?.querySelectorAll('dt') ?? []).map(
    (term) => `${term.textContent} — ${term.nextElementSibling?.textContent ?? ''}`,
  )

const answerSwitch = async (text: string) => {
  Array.from(findSwitchDialog()?.querySelectorAll('button') ?? [])
    .find((one) => one.textContent?.trim() === text)
    ?.click()
  await flushPromises()
}

const changePlanTo = async (wrapper: VueWrapper, name: string) => {
  await choosePlan(wrapper, name)
  await answerSwitch(`Switch to ${name}`)
}

describe('SubscriptionCard', () => {
  it('shows the title Numen Sync', async () => {
    const { wrapper } = await render(createResponse())

    expect(wrapper.text()).toContain('Numen Sync')
  })

  it('offers the plans to choose and shows no badge when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(wrapper.find('.subscription__badge').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('No subscription')
    expect(wrapper.text()).toContain(
      'Sync your vaults between your devices. Pick the storage you need, change it any time.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('has no badge and says until when the active subscription is paid', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.find('.subscription__badge').exists()).toBe(false)
    expect(getRow(wrapper, 'Renews')).toContain('December 23, 2026')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('shows the plan, the space used as a bar and the end of the paid period as rows', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(getRow(wrapper, 'Plan')).toContain('Plus, 100 GB, $15 / month')
    expect(getRow(wrapper, 'Storage')).toContain('Space used by your synced vaults')
    expect(getRow(wrapper, 'Storage')).toContain('2 GB of 100 GB')
    expect(wrapper.find('.island-card__aside').text()).not.toContain('Change plan')
    expect(await hasMenuItem(wrapper, 'Change plan')).toBe(true)
    expect(await hasMenuItem(wrapper, 'Manage subscription')).toBe(true)
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('2')
  })

  it('says when the changes stop in the grace phase and offers to buy', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
      }),
    )

    expect(wrapper.text()).toContain('Ended')
    expect(wrapper.text()).toContain(
      'The subscription ended on November 20, 2026. Changes are sent until November 27, 2026, then the vaults in the cloud can only be read.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('says the vaults can only be read and when the cloud deletes them, and offers to buy', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'readOnly',
        periodEndAt: '2026-11-13T12:00:00Z',
        writesEndAt: '2026-11-20T12:00:00Z',
        deleteAt: '2027-01-19T12:00:00Z',
      }),
    )

    expect(wrapper.text()).toContain('Read only')
    expect(wrapper.text()).toContain(
      'The subscription ended on November 13, 2026. The vaults in the cloud can only be read and changes stay on this device. The cloud deletes them on January 19, 2027.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('says the cloud deleted the vaults and offers to buy', async () => {
    const { wrapper } = await render(createResponse({ phase: 'deleted' }))

    expect(wrapper.text()).toContain('Deleted')
    expect(wrapper.text()).toContain(
      'The cloud deleted your vaults. Buy a subscription to sync again.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('asks for an identity and offers nothing when the device has none', async () => {
    const { wrapper } = await render(createResponse({ hasIdentity: false, phase: '' }))

    expect(wrapper.text()).toContain(
      'Sync your vaults between your devices. Notes and files are encrypted on your devices before they leave them. Create an identity in the card above to start.',
    )
    expect(wrapper.get('.subscription__badge').text()).toBe('Needs an identity')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('says the server cannot be reached and offers nothing', async () => {
    const client = createClient(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    )
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
    })
    await flushPromises()

    expect(wrapper.get('.subscription__badge').text()).toBe('Offline')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('opens the payment page in the browser and waits when the person buys', async () => {
    const { wrapper, client, openExternal } = await render(createResponse({ phase: 'none' }))

    await findChoose(wrapper, 'Plus')?.trigger('click')
    await flushPromises()

    expect(client.createCheckout).toHaveBeenCalledExactlyOnceWith('plus')
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
    expect(wrapper.get('[role="status"]').text()).toBe(
      'Finish the payment in your browser. This card updates by itself.',
    )
    expect(wrapper.text()).toContain('Pick the storage you need')
    expect(getChooseNames(wrapper)).toHaveLength(3)
    expect(getContent(wrapper).attributes('inert')).toBeDefined()
  })

  it('gives the plan rows back and stops waiting when the person cancels', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))
    await findChoose(wrapper, 'Plus')?.trigger('click')
    await flushPromises()

    await findCancel(wrapper)?.trigger('click')

    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(findCancel(wrapper)).toBeUndefined()
    expect(getContent(wrapper).attributes('inert')).toBeUndefined()
    expect(getChooseNames(wrapper)).toHaveLength(3)
  })

  it('shows each plan as a row to click with the name, the size and the price', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    const expected = [
      ['Basic', '10 GB of synced files', '$5', '/ month'],
      ['Plus', '100 GB of synced files', '$15', '/ month'],
      ['Pro', '250 GB of synced files', '$30', '/ month'],
    ] as const
    expect(wrapper.findAll('.subscription__row')).toHaveLength(expected.length)
    expected.forEach(([name, ...parts]) => {
      parts.forEach((part) => expect(getRow(wrapper, name)).toContain(part))
    })
    expect(getChooseNames(wrapper)).toStrictEqual([
      'Choose Basic, 10 GB, $5 / month',
      'Choose Plus, 100 GB, $15 / month',
      'Choose Pro, 250 GB, $30 / month',
    ])
    expect(wrapper.text()).not.toContain('Choose')
  })

  it('keeps the plan rows in place under the waiting layer while the payment page is being opened', async () => {
    const { wrapper, client, openExternal } = await render(createResponse({ phase: 'none' }))
    client.createCheckout.mockReturnValue(new Promise<string>(() => undefined))

    await findChoose(wrapper, 'Plus')?.trigger('click')
    await flushPromises()

    expect(getChooseNames(wrapper)).toHaveLength(3)
    expect(getContent(wrapper).attributes('inert')).toBeDefined()
    expect(wrapper.get('[role="status"]').text()).toBe(
      'Finish the payment in your browser. This card updates by itself.',
    )
    await findCancel(wrapper)?.trigger('click')
    expect(getContent(wrapper).attributes('inert')).toBeUndefined()
    expect(openExternal).not.toHaveBeenCalled()
  })

  it('shows the rows of an active subscription in the order plan, renewal, storage', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(getRowNames(wrapper)).toStrictEqual(['Plan', 'Renews', 'Storage'])
  })

  it('shows the plans by itself when the identity is created while the card is open', async () => {
    let answer = createResponse({ hasIdentity: false, phase: '' })
    const client = createClient(() => Promise.resolve(answer))
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
      refreshMs: { noIdentity: 50, ready: 60_000, unreachable: 60_000 },
    })
    await flushPromises()
    expect(wrapper.text()).toContain(
      'Sync your vaults between your devices. Notes and files are encrypted on your devices before they leave them. Create an identity in the card above to start.',
    )

    answer = createResponse({ phase: 'none' })
    await vi.waitFor(() => expect(findChoose(wrapper, 'Plus')).toBeDefined())

    expect(wrapper.text()).not.toContain('Create an identity')
  })

  it('says which server it could not reach and opens the settings from the card', async () => {
    const client = createClient(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    )
    const openSettings = vi.fn<() => void>()
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
      settings: {
        get: <T>() => 'https://sync.numen.md' as T,
        onChange: () => ({ dispose: () => undefined }),
      },
      openSettings,
      refreshMs: { noIdentity: 60_000, ready: 60_000, unreachable: 60_000 },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('https://sync.numen.md')
    expect(wrapper.text()).toContain('Start the server, or change the address in the settings.')
    expect(wrapper.text()).not.toContain('30 seconds')
    await wrapper
      .findAll('button')
      .find((one) => one.text() === 'Open the settings')
      ?.trigger('click')
    expect(openSettings).toHaveBeenCalledOnce()
  })

  it('offers every plan to choose with its name when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(findChoose(wrapper, 'Basic')).toBeDefined()
    expect(findChoose(wrapper, 'Plus')).toBeDefined()
    expect(findChoose(wrapper, 'Pro')).toBeDefined()
    expect(await hasMenuItem(wrapper, 'Change plan')).toBe(false)
  })

  it('lists every plan in the menu Change plan with its size and price, the current one checked and not to choose', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    await openPlans(wrapper)

    expect(getPlanChoices()).toStrictEqual(['Basic', 'Plus', 'Pro'])
    expect(
      getPlanItems().map((one) => one.querySelector('.menu__detail')?.textContent),
    ).toStrictEqual(['10 GB, $5 / month', '100 GB, $15 / month', '250 GB, $30 / month'])
    const current = getPlanItems().filter((one) => one.getAttribute('aria-checked') === 'true')
    expect(current.map(getPlanName)).toStrictEqual(['Plus'])
    expect(current[0]?.textContent).toContain('Current plan')
    expect(current[0]?.disabled).toBe(true)
    expect(current[0]?.querySelector('.menu__icon svg')).not.toBeNull()
    expect(getPlanItems().filter((one) => one.disabled)).toHaveLength(1)
  })

  it('asks before it changes the plan, with the plan now, the new plan, the charge and the next payment', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)

    await choosePlan(wrapper, 'Pro')

    expect(findSwitchDialog()?.querySelector('h2')?.textContent).toBe('Switch to Pro?')
    expect(getSwitchFacts()).toStrictEqual([
      'Now — Plus, 100 GB, $15 / month',
      'New plan — Pro, 250 GB, $30 / month',
      'Charged now — The difference for the rest of this period, to the card on file',
      'Next payment — $30 on December 23, 2026',
    ])
    expect(client.changePlan).not.toHaveBeenCalled()
    expect(getRow(wrapper, 'Plan')).toContain('Plus,')
  })

  it('keeps the plan and asks the server nothing when the switch is cancelled', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)
    await choosePlan(wrapper, 'Pro')

    await answerSwitch('Cancel')

    expect(findSwitchDialog()).toBeNull()
    expect(client.changePlan).not.toHaveBeenCalled()
    expect(getRow(wrapper, 'Plan')).toContain('Plus,')
  })

  it('does not ask to switch to the current plan', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    await choosePlan(wrapper, 'Plus')

    expect(findSwitchDialog()).toBeNull()
  })

  it('offers one Manage subscription and no plan to choose while the subscription is active', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(await hasMenuItem(wrapper, 'Manage subscription')).toBe(true)
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('changes the plan once the switch is confirmed, without opening a browser', async () => {
    const { wrapper, client, openExternal } = await render(ACTIVE_ON_PLUS)

    await changePlanTo(wrapper, 'Pro')

    expect(client.changePlan).toHaveBeenCalledExactlyOnceWith('pro')
    expect(openExternal).not.toHaveBeenCalled()
    expect(getRow(wrapper, 'Plan')).toContain('Pro, 250 GB, $30 / month')
    expect(findSwitchDialog()).toBeNull()
    expect(await hasMenuItem(wrapper, 'Change plan')).toBe(true)
  })

  it('shows the problem of a refused change and keeps the plan', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)
    client.changePlan.mockRejectedValue(
      Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
    )

    await changePlanTo(wrapper, 'Pro')

    expect(wrapper.find('[role="alert"]').text()).toContain('Could not reach the sync server')
    expect(getRow(wrapper, 'Plan')).toContain('100 GB, $15 / month')
  })

  it('shows why the server refused the change when the subscription reads the same', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)
    client.changePlan.mockRejectedValue(
      Object.assign(new Error('the account has no active subscription to change'), {
        code: 'failed_precondition',
      }),
    )

    await changePlanTo(wrapper, 'Pro')

    expect(wrapper.find('[role="alert"]').text()).toBe(
      'the account has no active subscription to change',
    )
    expect(getRow(wrapper, 'Plan')).toContain('Plus,')
  })

  it('shows the used space above the plan, says the cloud is full and offers a larger plan', async () => {
    const { wrapper } = await render(
      createResponse({
        ...ACTIVE_ON_PLUS,
        plan: 'basic',
        quotaBytes: 10 * GIB,
        usedBytes: 20 * GIB,
      }),
    )

    expect(getRow(wrapper, 'Storage')).toContain('20 GB of 10 GB')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('100')
    expect(wrapper.text()).toContain(
      'Cloud storage is full: changes stay on this device. Move to a larger plan to keep syncing.',
    )
    await openPlans(wrapper)
    expect(
      getPlanItems()
        .filter((one) => !one.disabled)
        .map(getPlanName),
    ).toStrictEqual(['Plus', 'Pro'])
  })

  it('does not say the cloud is full while there is room', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.text()).not.toContain('Cloud storage is full')
  })

  it('shows the plan of an ended subscription with the plans to choose and none to change', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
      }),
    )

    expect(getRow(wrapper, 'Plan')).toContain('Plus, 100 GB, $15 / month')
    expect(findChoose(wrapper, 'Plus')).toBeDefined()
    expect(await hasMenuItem(wrapper, 'Change plan')).toBe(false)
  })

  it('shows only the size of a plan the client does not know', async () => {
    const { wrapper } = await render(
      createResponse({ ...ACTIVE_ON_PLUS, plan: 'gold', quotaBytes: 50 * GIB }),
    )

    expect(wrapper.text()).toContain('50 GB')
    expect(wrapper.text()).not.toContain('gold')
  })
})

describe('SubscriptionCard trial and portal', () => {
  it('says the day the trial ends', async () => {
    const { wrapper } = await render(
      createResponse({ ...ACTIVE_ON_PLUS, isTrial: true, periodEndAt: '2026-12-07T12:00:00Z' }),
    )

    expect(getRow(wrapper, 'Trial until')).toContain('December 7, 2026')
    expect(wrapper.text()).not.toContain('Renews')
  })

  it('opens the portal in the browser with the item Manage subscription', async () => {
    const { wrapper, openExternal } = await render(ACTIVE_ON_PLUS)

    await toggleMenu(wrapper)
    findMenuItem('Manage subscription')?.click()
    await flushPromises()

    expect(openExternal).toHaveBeenCalledExactlyOnceWith(PORTAL_URL)
  })

  it('offers no portal to an identity without a subscription', async () => {
    const { wrapper } = await render(createResponse())

    expect(await hasMenuItem(wrapper, 'Manage subscription')).toBe(false)
  })
})
