/**
 * The transcript's words with a time in the gutter against each paragraph.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import TranscriptText from './TranscriptText.vue'
import { createTimeGutter } from '../model/timeGutter'

/* Two paragraphs that wrap over several rows in the measure the story sets, each with a time. */
const FIRST =
  'Last week we left the demon at the door, letting the fast molecules through one way and the ' +
  'slow ones the other. After a while one side is hot and the other is cold, and nothing was ' +
  'pushed and nothing was heated.'
const SECOND =
  'В сарае на краю участка лежат лопаты, лейки и мешок с семенами. Ключ от двери висит на ' +
  'гвозде, а у порога стоит старая тачка.'
const TIMED = [FIRST, '', SECOND].join('\n')
const TIMED_TIMES = ['0:01', '', '0:41']

const meta: Meta = {
  title: 'Recording/Transcript',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/**
 * One time is set against each timed paragraph, however many rows the paragraph wraps over.
 */
export const Timed: Story = {
  render: () => ({
    components: { TranscriptText },
    setup: () => {
      const times = createTimeGutter(() => undefined)
      times.show({ times: TIMED_TIMES, current: 0, isFollowing: false })
      return { text: TIMED, extensions: times.extension }
    },
    template: `<TranscriptText :model-value="text" :is-editable="true" label="Transcript" :extensions="extensions" style="--editor-measure: 24rem" class="numen h-screen bg-surface" />`,
  }),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('.cm-time')).not.toBeNull())
    const times = Array.from(canvasElement.querySelectorAll('.cm-time')).map((one) => one.textContent)
    await expect(times).toEqual(TIMED_TIMES.filter((text) => text !== ''))
  },
}
