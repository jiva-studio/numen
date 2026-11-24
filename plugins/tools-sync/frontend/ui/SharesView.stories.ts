import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import type {
  FolderSharesClient,
  ReceivedShare,
  ReceivedSharesClient,
  SharedFolder,
} from '../api/shares'
import { createSharesPage } from './sharesPage'
import { frameDecorator } from './storyFrame'
import '../styles.css'

const OWNER = 'EEEE FFFF GGGG HHHH IIII'

const food = (state: ReceivedShare['state']): ReceivedShare => ({
  vaultId: 'r1',
  name: 'Food',
  ownerFingerprint: OWNER,
  state,
  lastSyncedAt: '',
})

const RECIPES: SharedFolder = {
  vaultId: 'v1',
  folder: 'Recipes',
  state: 'active',
  readers: [{ label: 'Phone', fingerprint: '1111 2222 3333 4444', state: 'active' }],
}

interface Setup {
  readonly given?: SharedFolder[]
  readonly received?: ReceivedShare[]
}

const createPage = (setup: Setup) => {
  const given: FolderSharesClient = {
    listShares: () => Promise.resolve(setup.given ?? []),
    removeReader: () => Promise.resolve(),
    stopShare: () => Promise.resolve(),
  }
  const received: ReceivedSharesClient = {
    listReceived: () => Promise.resolve(setup.received ?? []),
    leaveShare: () => Promise.resolve(),
  }
  return createSharesPage({ given, received, refreshMs: 1_000_000 })
}

const meta = {
  title: 'NumenSync/SharedFolders',
  decorators: [frameDecorator],
  render: (args) => ({
    components: { Page: args.page },
    template: '<Page />',
  }),
} satisfies Meta<{ page: ReturnType<typeof createSharesPage> }>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
  args: { page: createPage({}) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('No folder is shared yet')).toBeVisible()
  },
}

export const SharedByYou: Story = {
  args: { page: createPage({ given: [RECIPES] }) },
  play: async ({ canvasElement }) => {
    const list = await within(canvasElement).findByRole('list', { name: 'Shared by you' })
    await expect(within(list).getByText('Recipes')).toBeVisible()
    await expect(within(list).getByText('Device «Phone» (1111 2222 3333…)')).toBeVisible()
    await expect(within(list).getByRole('button', { name: 'Remove access' })).toBeVisible()
    await expect(within(list).getByRole('button', { name: 'Actions for Recipes' })).toBeVisible()
  },
}

export const SharedWithYou: Story = {
  args: { page: createPage({ received: [food('ready')] }) },
  play: async ({ canvasElement }) => {
    const list = await within(canvasElement).findByRole('list', { name: 'Shared with you' })
    await expect(within(list).getByText('Up to date')).toBeVisible()
    await expect(within(list).getByText('Read only · Owner EEEE FFFF GGGG…')).toBeVisible()
  },
}

export const Receiving: Story = {
  args: { page: createPage({ received: [food('receiving')] }) },
  play: async ({ canvasElement }) => {
    await expect(await within(canvasElement).findByText('Getting the folder…')).toBeVisible()
  },
}

export const Ended: Story = {
  args: { page: createPage({ received: [food('ended')] }) },
  play: async ({ canvasElement }) => {
    await expect(
      await within(canvasElement).findByText('The person stopped sharing this folder'),
    ).toBeVisible()
  },
}
