/**
 * The window going with notes still unwritten.
 *
 * It overlays the work because nothing else the user does can dismiss it, and
 * how a name too long for the row affects the three actions is browser
 * behavior.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import UnsavedChangesPrompt from './UnsavedChangesPrompt.vue'
import type { ConflictPrompt } from '../model/flush'

/** The strings the window passes to the prompt. The component defines none of its own. */
const WORDS: InstanceType<typeof UnsavedChangesPrompt>['$props']['words'] = {
  going: 'Saving paused: the files of these notes changed on disk.',
  keep: 'Keep my version',
  take: 'Use disk version',
  later: 'Not yet',
}

const UNBROKEN = `${'A note whose name nobody shortened and which runs on past '.repeat(8)}.md`

const conflict = (path: string): ConflictPrompt => ({
  note: path,
  keep: fn(() => Promise.resolve()),
  take: fn(() => Promise.resolve()),
  later: fn(),
})

const meta = {
  title: 'Window/Unsaved Changes',
  component: UnsavedChangesPrompt,
  parameters: { layout: 'fullscreen' },
  args: { called: (path: string) => path.split('/').pop() ?? path, words: WORDS },
} satisfies Meta<typeof UnsavedChangesPrompt>

export default meta
type Story = StoryObj<typeof meta>

const notice = (canvas: HTMLElement) => within(canvas).queryByRole('alertdialog')

const notes = (canvas: HTMLElement) => within(canvas).getAllByRole('listitem')

const getAnswers = (canvas: HTMLElement) => within(canvas).getAllByRole('button')

/** Two notes, each with the three ways out of it. */
export const TwoNotes: Story = {
  args: { conflicts: [conflict('physics/Entropy.md'), conflict('Heat.md')] },
  play: async ({ canvasElement }) => {
    const response = notice(canvasElement)
    await expect(response).not.toBeNull()
    await expect(notes(canvasElement)).toHaveLength(2)
    await expect(getAnswers(canvasElement)).toHaveLength(6)

    // It overlays the work, at the bottom of the window and inside it.
    const box = response!.getBoundingClientRect()
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
  },
}

/** One note, answered. */
export const ResolvedForOneNote: Story = {
  args: { conflicts: [conflict('physics/Entropy.md'), conflict('Heat.md')] },
  play: async ({ args, canvasElement }) => {
    const [first, second] = args.conflicts

    // The response applies to the note it is shown beside, and the other note is
    // left unchanged.
    const keep = getAnswers(canvasElement).find((one) => one.textContent?.trim() === WORDS.keep)
    await userEvent.click(keep as HTMLElement)
    await waitFor(() => expect(first?.keep).toHaveBeenCalledTimes(1))
    await expect(first?.take).not.toHaveBeenCalled()
    await expect(second?.keep).not.toHaveBeenCalled()
  },
}

export const AnUnbrokenName: Story = {
  args: { conflicts: [conflict(UNBROKEN)] },
  play: async ({ canvasElement }) => {
    const row_ = notes(canvasElement)[0]!
    await expect(within(row_).getByText(UNBROKEN.split('/').pop()!)).toBeVisible()

    // The answers stay on the row.
    for (const answer of getAnswers(canvasElement)) await expect(answer).toBeVisible()
  },
}

/** Nothing unwritten, so nothing is drawn and nothing is in the way. */
export const NothingUnwritten: Story = {
  args: { conflicts: [] },
  play: async ({ canvasElement }) => {
    await expect(notice(canvasElement)).toBeNull()
  },
}
