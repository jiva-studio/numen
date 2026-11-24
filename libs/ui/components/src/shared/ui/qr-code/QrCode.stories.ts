/**
 * The text of a link drawn as a QR code another device reads with its camera.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import QrCode from './QrCode.vue'

const meta = {
  title: 'QrCode',
  component: QrCode,
  parameters: { layout: 'centered' },
} satisfies Meta<typeof QrCode>

export default meta
type Story = StoryObj<typeof meta>

/** The text of a link to join an identity. */
export const Playground: Story = {
  args: {
    text: 'numen://link?v=1&c=AAAAAAAAAAAAAAAAAAAAAA&k=BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB&s=CCCCCCCCCCCCCCCCCCCCCC',
  },
}

/** A short text. */
export const Short: Story = { args: { text: 'numen', size: 140 } }
