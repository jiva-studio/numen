/**
 * Every situation a switch has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * A switch draws no words of its own, so what is awkward about it is the name
 * beside it and the keyboard reaching it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Switch from './Switch.vue'
import { lightness } from '@/shared/fixtures/colour'
import { DARK, drawnDark } from '@/shared/fixtures/theme'

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  on: boolean
  disabled: boolean
  /** What the switch is called, said beside it. */
  said: string
}

const meta: Meta<Knobs> = {
  title: 'Controls/Switch',
  component: Switch,
  parameters: { layout: 'centered' },
  argTypes: {
    on: { control: 'boolean' },
    disabled: { control: 'boolean' },
    said: { control: 'text' },
  },
  args: { on: true, disabled: false, said: 'Show the grid' },
  render: (args) => ({
    components: { Switch },
    setup: () => {
      const on = ref(args.on)
      return { args, on }
    },
    // A switch is not a form control, so a label around it names nothing: the
    // words beside it are pointed at instead, which is how a window does it.
    template: `
      <div style="display: flex; align-items: center; gap: 0.625rem; padding: 2rem; inline-size: 20rem">
        <Switch v-model="on" :disabled="args.disabled" aria-labelledby="switch-said" />
        <span id="switch-said" style="min-inline-size: 0; overflow-wrap: anywhere">{{ args.said }}</span>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const switched = (canvas: HTMLElement): HTMLElement => {
  const found = canvas.querySelector<HTMLElement>('[role="switch"]')
  if (!found) throw new Error('no switch')
  return found
}

/** A switch that is on. */
export const ASwitch: Story = {}

/** A switch that is off. */
export const Off: Story = { args: { on: false } }

/** A switch nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** A name in another script. */
export const OtherScripts: Story = { args: { said: 'Показывать сетку' } }

/** A name far longer than anything a setting is called. */
export const FarTooLong: Story = {
  args: { said: 'Show the grid behind everything drawn on the canvas below it '.repeat(3) },
}

/** A name with nothing in it to break at. */
export const Unbroken: Story = { args: { said: UNBROKEN } }

/**
 * No name at all. The keyboard rule is off here because this is a switch with
 * nothing to read out, which is the whole of what the story draws.
 */
export const NoTextAtAll: Story = { args: { said: '' }, parameters: { reach: false } }

/**
 * The switch on the dark set of tokens. Which way it stands is told by the
 * colour of the track, so the two are held apart there as well.
 */
export const Dark: Story = {
  globals: DARK,
  args: { on: false },
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)
    const control = switched(canvasElement)
    const off = getComputedStyle(control).backgroundColor

    await userEvent.click(control)
    await waitFor(() => expect(control.getAttribute('aria-checked')).toBe('true'))
    await waitFor(() => expect(getComputedStyle(control).backgroundColor).not.toBe(off))

    // The accent the track comes to rest in stands above the quiet colour it
    // was, which is the way round the dark set is written.
    await waitFor(() =>
      expect(lightness(getComputedStyle(control).backgroundColor)).toBeGreaterThan(
        lightness(off) + 8,
      ),
    )
  },
}

/** It is announced as a switch, and says which way it is. */
export const AnnouncedAsASwitch: Story = {
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)
    expect(control.getAttribute('role')).toBe('switch')
    expect(control.getAttribute('aria-checked')).toBe('true')
  },
}

/**
 * What a person actually sees: the thumb slides across the track and the track
 * fills behind it. Neither is a state a test can read off an attribute, so
 * both are measured where they are drawn.
 */
export const TheThumbSlidesAndTheTrackFills: Story = {
  args: { on: false },
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)
    const thumb = control.firstElementChild as HTMLElement
    const across = () => thumb.getBoundingClientRect().left - control.getBoundingClientRect().left
    const filling = () => getComputedStyle(control).backgroundColor

    const wasAcross = across()
    const wasFilling = filling()

    await userEvent.click(control)
    await waitFor(() => expect(control.getAttribute('aria-checked')).toBe('true'))

    // The thumb has moved the width of a thumb, and it is still on the track.
    await waitFor(() => expect(across()).toBeGreaterThan(wasAcross + thumb.offsetWidth / 2))
    const track = control.getBoundingClientRect()
    const box = thumb.getBoundingClientRect()
    expect(box.right).toBeLessThanOrEqual(track.right + 1)
    expect(box.left).toBeGreaterThanOrEqual(track.left - 1)

    // And the track behind it is not the colour it was.
    await waitFor(() => expect(filling()).not.toBe(wasFilling))

    await userEvent.click(control)
    await waitFor(() => expect(across()).toBeCloseTo(wasAcross, 0))
    await waitFor(() => expect(filling()).toBe(wasFilling))
  },
}

/** The space bar turns it, and the ring says the keyboard is on it. */
export const TheSpaceBarTurnsIt: Story = {
  args: { on: false },
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)

    await userEvent.tab()
    expect(document.activeElement).toBe(control)
    expect(getComputedStyle(control).boxShadow).not.toBe('none')

    await userEvent.keyboard(' ')
    expect(control.getAttribute('aria-checked')).toBe('true')

    await userEvent.keyboard(' ')
    expect(control.getAttribute('aria-checked')).toBe('false')
  },
}

/** The keyboard goes nowhere near a switch nobody may turn. */
export const DisabledTakesNoKeyboard: Story = {
  args: { disabled: true },
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)
    await userEvent.tab()
    expect(document.activeElement).not.toBe(control)
  },
}
