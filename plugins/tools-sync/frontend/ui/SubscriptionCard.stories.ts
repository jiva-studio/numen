import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { asValue } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'
import '../styles.css'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

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
    await expect(await canvas.findByText('Choose a plan')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Choose Plus' })).toBeVisible()
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
    await expect(await canvas.findByRole('button', { name: 'Choose Plus' })).toBeVisible()
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
    await expect(await canvas.findByRole('button', { name: 'Choose Plus' })).toBeVisible()
  },
}

export const Deleted: Story = {
  args: createArgs(() => Promise.resolve(createResponse({ phase: 'deleted' }))),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Deleted')).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Choose Plus' })).toBeVisible()
  },
}

export const NoIdentity: Story = {
  args: createArgs(() => Promise.resolve(createResponse({ hasIdentity: false, phase: '' }))),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByText('Create an identity to buy a subscription.')).toBeVisible()
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
    await expect(await canvas.findByText('Could not reach the sync server')).toBeVisible()
    await expect(await canvas.findByText(/https:\/\/sync\.numen\.md/)).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Open the settings' })).toBeVisible()
    await expect(canvas.queryByRole('button', { name: /^Choose / })).toBeNull()
  },
}

export const OffersEveryPlan: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('button', { name: 'Choose Basic' })).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Choose Plus' })).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Choose Pro' })).toBeVisible()
    await expect(await canvas.findByText('$5 / month')).toBeVisible()
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
    await expect(await canvas.findByText('Plus, 100 GB')).toBeVisible()
    await expect(await canvas.findByText('2 GB of 100 GB used')).toBeVisible()
    await expect(await canvas.findByRole('progressbar', { name: 'Space used' })).toBeVisible()
    await expect(await canvas.findByRole('button', { name: 'Change plan' })).toBeVisible()
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
    await expect(await canvas.findByText('20 GB of 10 GB used')).toBeVisible()
    await expect(
      await canvas.findByText('Cloud storage is full: changes stay on this device'),
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
    await expect(await canvas.findByText('Plus, 100 GB')).toBeVisible()
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
