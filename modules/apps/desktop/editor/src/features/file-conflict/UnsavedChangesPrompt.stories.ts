/**
 * The window going with notes still unwritten.
 *
 * It sits over the work because nothing else the person does can end it, and
 * what a name too long for the row does to the three ways out of it is the
 * browser's answer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import UnsavedChangesPrompt from './UnsavedChangesPrompt.vue'
import type { ConflictPrompt } from './flush'
import type { UnsavedChangesWords } from './UnsavedChangesPrompt.vue'

/** The words a window hands the prompt. The component knows none of its own. */
const WORDS: UnsavedChangesWords = {
  going: 'These notes stopped saving because their files changed. The window waits.',
  keep: 'Keep mine',
  take: "Take the file's",
  later: 'Not yet',
}

const UNBROKEN = `${'A note whose name nobody shortened and which runs on past '.repeat(8)}.md`

/** One note standing, with the three answers written down as they are given. */
const conflict = (path: string): ConflictPrompt => ({
  note: path,
  keep: fn(async () => {}),
  take: fn(async () => {}),
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

/** What stands in the way of the window going, if anything does. */
const notice = (canvas: HTMLElement) => within(canvas).queryByRole('alertdialog')

/** The rows, one per note still unwritten. */
const notes = (canvas: HTMLElement) => within(canvas).getAllByRole('listitem')

/** Every way out of every note standing. */
const answers = (canvas: HTMLElement) => within(canvas).getAllByRole('button')

/** Two notes, each with the three ways out of it. */
export const TwoNotes: Story = {
  args: { conflicts: [conflict('physics/Entropy.md'), conflict('Heat.md')] },
  play: async ({ canvasElement }) => {
    const said = notice(canvasElement)
    await expect(said).not.toBeNull()
    await expect(notes(canvasElement)).toHaveLength(2)
    await expect(answers(canvasElement)).toHaveLength(6)

    // An answer is a word in the sentence it stands in, with a line under it.
    const answer = getComputedStyle(answers(canvasElement)[0]!)
    await expect(answer.textDecorationLine).toBe('underline')
    await expect(answer.backgroundColor).toBe('rgba(0, 0, 0, 0)')

    // It sits over the work, at the foot of the window and inside it.
    const box = said!.getBoundingClientRect()
    await expect(getComputedStyle(said!).position).toBe('fixed')
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
  },
}

/** One note, answered. */
export const AnsweredForOneNote: Story = {
  args: { conflicts: [conflict('physics/Entropy.md'), conflict('Heat.md')] },
  play: async ({ args, canvasElement }) => {
    const [first, second] = args.conflicts

    // The answer given is the answer for the note it stands beside, and the
    // other note is left standing.
    const keep = answers(canvasElement).find((one) => one.textContent?.trim() === WORDS.keep)
    await userEvent.click(keep as HTMLElement)
    await waitFor(() => expect(first?.keep).toHaveBeenCalledTimes(1))
    await expect(first?.take).not.toHaveBeenCalled()
    await expect(second?.keep).not.toHaveBeenCalled()
  },
}

/** A name far longer than the row it stands in. */
export const AnUnbrokenName: Story = {
  args: { conflicts: [conflict(UNBROKEN)] },
  play: async ({ canvasElement }) => {
    const row_ = notes(canvasElement)[0]!
    const title = within(row_).getByText(UNBROKEN.split('/').pop()!)

    // The name is cut short, and the answers stay on the row.
    await expect(title.scrollWidth).toBeGreaterThan(title.clientWidth)

    const row = row_.getBoundingClientRect()
    for (const answer of answers(canvasElement)) {
      const box = answer.getBoundingClientRect()
      await expect(box.right).toBeLessThanOrEqual(Math.ceil(row.right))
      await expect(box.width).toBeGreaterThan(0)
    }
  },
}

/** Nothing unwritten, so nothing is drawn and nothing is in the way. */
export const NothingUnwritten: Story = {
  args: { conflicts: [] },
  play: async ({ canvasElement }) => {
    await expect(notice(canvasElement)).toBeNull()
  },
}
