/**
 * The popover of the version chip in the title bar, open. Also the test corpus: each story is run in
 * a browser by `@storybook/addon-vitest`.
 *
 * The popover is drawn below the chip, framed by a border on a ground of the theme, and holds the
 * details of the release above a row of three buttons; only a browser lays that out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import type { Update } from './api/updater'
import UpdateIndicator from './UpdateIndicator.vue'
import { WORDS as words } from './words'
import './styles.css'

/** A made-up release with a page of notes apart from the download. */
const RELEASE: Update = {
  version: '2026.12.1',
  build: 412,
  channel: 'stable',
  releasedAt: '2026-11-20T09:00:00Z',
  notesUrl: 'https://example.invalid/releases/2026.12.1',
  downloadUrl: 'https://example.invalid/download/2026.12.1',
}

const meta = {
  title: 'Update notifier/Popover',
  component: UpdateIndicator,
  parameters: { layout: 'fullscreen' },
  args: { version: '2026.11.2', update: RELEASE, platform: 'linux', onDismiss: fn() },
  render: (args) => ({
    components: { UpdateIndicator },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;align-items:flex-start;justify-content:flex-end;block-size:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <header style="display:flex;align-items:center;justify-content:flex-end;position:relative;inline-size:100%;block-size:2rem;background:var(--numen-titlebar);border-bottom:1px solid var(--numen-rule);font-size:var(--numen-text-1)">
          <UpdateIndicator v-bind="args" />
        </header>
      </div>
    `,
  }),
} satisfies Meta<typeof UpdateIndicator>

export default meta
type Story = StoryObj<typeof meta>

const isApart = (first: DOMRect, second: DOMRect): boolean =>
  second.top >= first.bottom || second.left >= first.right

const getGaps = (row: readonly DOMRect[]): number[] =>
  row.slice(1).map((one, index) => one.left - (row[index]?.right ?? 0))

/** The popover is below the title bar, inside the window, framed, with its details and buttons laid out. */
export const PopoverOpened: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const indicator = await canvas.findByRole('button', { name: /^Update available: v/ })
    await expect(indicator).toHaveTextContent(`v${RELEASE.version}`)
    await userEvent.click(indicator)

    const popover = await canvas.findByRole('dialog', { name: words.title })
    const bar = canvasElement.querySelector('header') as HTMLElement
    await waitFor(() =>
      expect(popover.getBoundingClientRect().top).toBeGreaterThanOrEqual(
        bar.getBoundingClientRect().bottom,
      ),
    )
    const box = popover.getBoundingClientRect()
    await expect(box.left).toBeGreaterThanOrEqual(0)
    await expect(box.top).toBeGreaterThanOrEqual(0)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
    const trigger = indicator.getBoundingClientRect()
    await expect(box.left).toBeLessThan(trigger.right)
    await expect(box.right).toBeGreaterThan(trigger.left)

    const style = getComputedStyle(popover)
    await expect(style.backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    await expect(style.borderTopStyle).toBe('solid')
    await expect(style.borderTopWidth).toBe('1px')

    const header = popover.querySelector('header') as HTMLElement
    await expect(header).toHaveTextContent(words.title)
    await expect(header).not.toHaveTextContent(RELEASE.version)
    const release = popover.querySelector('.update-popover__release') as HTMLElement
    await expect(release).toHaveTextContent(
      new RegExp(`^v${RELEASE.version} · ${words.releasedOn}`),
    )
    await expect(release).toBeVisible()
    const title = within(popover).getByText(words.title, { exact: true })
    await expect(isApart(title.getBoundingClientRect(), release.getBoundingClientRect())).toBe(true)

    const boxes: DOMRect[] = []
    for (const name of [words.download, words.releaseNotes, words.dismiss]) {
      const button = within(popover).getByRole('button', { name })
      await expect(button).toBeVisible()
      await expect(button.tagName).toBe('BUTTON')
      await expect(Number.parseFloat(getComputedStyle(button).paddingLeft)).toBeGreaterThan(0)
      const one = button.getBoundingClientRect()
      await expect(one.left).toBeGreaterThanOrEqual(box.left)
      await expect(one.right).toBeLessThanOrEqual(box.right)
      await expect(one.bottom).toBeLessThanOrEqual(box.bottom)
      boxes.push(one)
    }
    await expect(
      Math.max(...boxes.map((one) => one.top)) - Math.min(...boxes.map((one) => one.top)),
    ).toBeLessThan(4)
    await expect(getGaps(boxes).every((gap) => gap > 0)).toBe(true)
    await expect(within(popover).getByRole('button', { name: /^Close/ })).toBeVisible()

    await userEvent.keyboard('{Escape}')

    await waitFor(() => expect(canvas.queryByRole('dialog')).toBeNull())
  },
}
