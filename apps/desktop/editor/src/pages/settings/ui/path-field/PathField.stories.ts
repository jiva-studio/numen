/**
 * The path field of a file setting, with one path and with several, rendered in a browser in both
 * token sets. The checks cover what jsdom cannot: the actions are the library's outline buttons, of
 * one height and one border color, and that border is the field border token, not the text color.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import SettingRow from '../setting-row/SettingRow.vue'
import { WORDS } from '../../words'
import PathField from './PathField.vue'

interface Knobs {
  path: string
  isMulti: boolean
  problem: string
}

const onChoose = fn()
const onClear = fn()

const render = (args: Knobs) => ({
  components: { PathField, SettingRow },
  setup: () => ({ args, words: WORDS, onChoose, onClear }),
  template: `
    <div class="numen" style="container:settings / inline-size;inline-size:48rem;padding:1rem;background:var(--numen-surface)">
      <SettingRow v-slot="{ labelledBy }" at="path" name="Programs" detail="The programs this plugin runs.">
        <PathField
          :path="args.path"
          :problem="args.problem"
          :labelled-by="labelledBy"
          :is-multi="args.isMulti"
          :words="words"
          @choose="onChoose"
          @clear="onClear"
        />
      </SettingRow>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/Path field',
  component: PathField,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    path: { control: 'text' },
    isMulti: { control: 'boolean' },
    problem: { control: 'text' },
  },
  args: { path: '/opt/sample/bin/program', isMulti: false, problem: '' },
  render,
}

export default meta
type Story = StoryObj<Knobs>

const getField = (canvasElement: HTMLElement) => {
  const field = canvasElement.querySelector<HTMLElement>('.settings__path')
  if (!field) throw new Error('the path field is not drawn')
  return within(field)
}

const expectOneButtonFamily = async (canvasElement: HTMLElement) => {
  const buttons = getField(canvasElement).getAllByRole('button')
  await expect(buttons.length).toBeGreaterThan(1)
  for (const button of buttons) {
    await expect(button).toHaveAttribute('data-slot', 'button')
  }
}

/** One path: Choose and Clear. */
export const OnePath: Story = {
  play: async ({ canvasElement }) => {
    const canvas = getField(canvasElement)
    await expectOneButtonFamily(canvasElement)
    const choose = canvas.getByRole('button', { name: WORDS.pathChoose })
    const clear = canvas.getByRole('button', { name: WORDS.pathClear })

    await userEvent.click(clear)
    await expect(onClear).toHaveBeenLastCalledWith('/opt/sample/bin/program')
    await userEvent.click(choose)
    await expect(onChoose).toHaveBeenCalled()
  },
}

/** No path: Choose alone, and no Clear. */
export const NoPath: Story = {
  args: { path: '' },
  play: async ({ canvasElement }) => {
    const canvas = getField(canvasElement)
    await expect(canvas.getAllByRole('button').map((one) => one.textContent?.trim())).toStrictEqual(
      [WORDS.pathChoose],
    )
  },
}

/** Several paths: Add above, and a Clear on each row in the same button family. */
export const SeveralPaths: Story = {
  args: {
    path: '/opt/sample/bin/first\n/opt/sample/a-folder-name-far-too-long-to-fit/bin/second',
    isMulti: true,
  },
  play: async ({ canvasElement }) => {
    const canvas = getField(canvasElement)
    await expectOneButtonFamily(canvasElement)
    await expect(canvas.getByRole('button', { name: WORDS.pathAdd })).toBeVisible()
    const clears = canvas.getAllByRole('button', { name: WORDS.pathClear })
    await expect(clears).toHaveLength(2)
    await userEvent.click(clears[1]!)
    await expect(onClear).toHaveBeenLastCalledWith(
      '/opt/sample/a-folder-name-far-too-long-to-fit/bin/second',
    )
  },
}

/** One path on the dark token set. */
export const OnePathInDark: Story = {
  globals: { theme: 'dark' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.ownerDocument.documentElement.dataset['theme']).toBe('dark')
    await expectOneButtonFamily(canvasElement)
  },
}

/** Several paths on the dark token set. */
export const SeveralPathsInDark: Story = {
  ...SeveralPaths,
  globals: { theme: 'dark' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.ownerDocument.documentElement.dataset['theme']).toBe('dark')
    await expectOneButtonFamily(canvasElement)
  },
}
