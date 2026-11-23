import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { asValue } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'
import '../styles.css'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const CHOOSE_PLUS = 'Choose Plus, 100 GB, $15 / month'
const NOW = new Date('2026-11-23T12:00:00Z')

const GIB = 2 ** 30
const PLANS = [
  { id: 'basic', name: 'Basic', quotaBytes: 10 * GIB, priceCents: 500, currency: 'USD' },
  { id: 'plus', name: 'Plus', quotaBytes: 100 * GIB, priceCents: 1500, currency: 'USD' },
  { id: 'pro', name: 'Pro', quotaBytes: 250 * GIB, priceCents: 3000, currency: 'USD' },
]

const createResponse = (patch: Partial<SubscriptionResponse>): SubscriptionResponse => ({
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

const createArgs = (answer: () => Promise<SubscriptionResponse>) => ({
  client: {
    getSubscription: answer,
    listPlans: () => Promise.resolve(PLANS),
    changePlan: (plan: string) =>
      Promise.resolve(createResponse({ phase: 'active', plan, quotaBytes: 100 * GIB })),
    createCheckout: () => Promise.resolve('https://pay.example/checkout/abc'),
    createPortalSession: () => Promise.resolve('https://portal.example/session/1'),
  },
  openExternal: () => Promise.resolve(asValue(null)),
  now: () => NOW,
})

const meta = {
  title: 'NumenSync/Subscription',
  component: SubscriptionCard,
  args: createArgs(() => Promise.resolve(createResponse({}))),
} satisfies Meta<typeof SubscriptionCard>

export default meta
type Story = StoryObj<typeof meta>

export const None: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText(/^Sync your vaults between your devices/)).toBeVisible()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
    await expect(canvas.queryByText('No subscription')).toBeNull()
  },
}

export const Active: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'active',
        periodEndAt: '2026-12-23T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
        usedBytes: 2 * GIB,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('December 23, 2026')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Change plan' })).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Manage subscription' })).toBeVisible()
    await expect(canvas.queryByRole('button', { name: /^Choose / })).toBeNull()
  },
}

export const Grace: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Ended')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
  },
}

export const ReadOnly: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'readOnly',
        periodEndAt: '2026-11-13T12:00:00Z',
        writesEndAt: '2026-11-20T12:00:00Z',
        deleteAt: '2027-01-19T12:00:00Z',
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Read only')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
  },
}

export const Deleted: Story = {
  args: createArgs(() => Promise.resolve(createResponse({ phase: 'deleted' }))),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Deleted')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
  },
}

export const NoIdentity: Story = {
  args: createArgs(() => Promise.resolve(createResponse({ hasIdentity: false, phase: '' }))),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByText(
        'Sync your vaults between your devices. Notes and files are encrypted on your devices before they leave them. Create an identity in the card above to start.',
      ),
    ).toBeVisible()
    await expect(canvas.getByText('Needs an identity')).toBeVisible()
    await expect(canvas.queryByRole('button', { name: /^Choose / })).toBeNull()
  },
}

export const Unreachable: Story = {
  args: {
    ...createArgs(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    ),
    settings: {
      get: <T>() => 'https://sync.numen.md' as T,
      onChange: () => ({ dispose: () => undefined }),
    },
    openSettings: () => undefined,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Offline')).toBeVisible()
    await expect(await canvas.findByText(/https:\/\/sync\.numen\.md/)).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Open the settings' })).toBeVisible()
    await expect(canvas.queryByRole('button', { name: /^Choose / })).toBeNull()
  },
}

export const OffersEveryPlan: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(
      await canvas.findByRole('button', { name: 'Choose Basic, 10 GB, $5 / month' }),
    ).toBeVisible()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
    await expect(
      await canvas.findByRole('button', { name: 'Choose Pro, 250 GB, $30 / month' }),
    ).toBeVisible()
    await expect(await canvas.findByText('10 GB of synced files')).toBeVisible()
  },
}

export const ActiveWithPlanAndUsage: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'active',
        periodEndAt: '2026-12-23T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
        usedBytes: 2 * GIB,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('100 GB, $15 / month')).toBeVisible()
    await expect(await canvas.findByText('2 GB of 100 GB')).toBeVisible()
    await expect(await canvas.findByRole('progressbar', { name: 'Space used' })).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Change plan' })).toBeVisible()
  },
}

export const ChangePlanOpen: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'active',
        periodEndAt: '2026-12-23T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
        usedBytes: 2 * GIB,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'Change plan' }))
    const page = within(canvasElement.ownerDocument.body)
    const plans = await page.findAllByRole('menuitemradio')
    await expect(plans).toHaveLength(3)
    const current = plans.filter((one) => one.getAttribute('aria-checked') === 'true')
    await expect(current).toHaveLength(1)
    await expect(current[0]).toHaveTextContent('Current plan')
    await expect(current[0]).toBeDisabled()
  },
}

export const ConfirmSwitch: Story = {
  args: ChangePlanOpen.args,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: 'Change plan' }))
    const page = within(canvasElement.ownerDocument.body)
    await userEvent.click(await page.findByRole('menuitemradio', { name: /^Pro/ }))
    const dialog = await page.findByRole('alertdialog', { name: 'Switch to Pro?' })
    await expect(dialog).toHaveTextContent('Plus, 100 GB, $15 / month')
    await expect(dialog).toHaveTextContent('Pro, 250 GB, $30 / month')
    await expect(dialog).toHaveTextContent('$30 on December 23, 2026')
    await expect(within(dialog).getByRole('button', { name: 'Switch to Pro' })).toBeVisible()
    await expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus()
  },
}

export const FullPlan: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'active',
        periodEndAt: '2026-12-23T12:00:00Z',
        plan: 'basic',
        quotaBytes: 10 * GIB,
        usedBytes: 20 * GIB,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('20 GB of 10 GB')).toBeVisible()
    await expect(
      await canvas.findByText(/^Cloud storage is full: changes stay on this device/),
    ).toBeVisible()
  },
}

export const EndedWithPlan: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Plus, 100 GB, $15 / month')).toBeVisible()
    await expect(canvas.queryByRole('button', { name: 'Change plan' })).toBeNull()
  },
}

export const Trial: Story = {
  args: createArgs(() =>
    Promise.resolve(
      createResponse({
        phase: 'active',
        periodEndAt: '2026-12-07T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
        isTrial: true,
      }),
    ),
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Trial until')).toBeVisible()
    await expect(await canvas.findByText('December 7, 2026')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Manage subscription' })).toBeVisible()
  },
}

export const WaitingForPayment: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: CHOOSE_PLUS }))
    await expect(await canvas.findByRole('status')).toHaveTextContent(
      'Finish the payment in your browser. This card updates by itself.',
    )
    await expect(await canvas.findByRole('button', { name: 'Cancel' })).toBeVisible()
    const plus = canvas.getByText('100 GB of synced files')
    await expect(plus).toBeVisible()
    await expect(plus.closest('[inert]')).not.toBeNull()
  },
}

export const CancelWaiting: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await canvas.findByRole('button', { name: CHOOSE_PLUS }))
    await userEvent.click(await canvas.findByRole('button', { name: 'Cancel' }))
    await expect(canvas.queryByRole('status')).toBeNull()
    await expect(await canvas.findByRole('button', { name: CHOOSE_PLUS })).toBeVisible()
  },
}
