/**
 * The contrast of every control that draws text or a mark on a solid ground,
 * in every variant, size and state, in the light set and in the dark one.
 *
 * Also the test corpus: a browser resolves each colour as drawn, the ground
 * with the translucent layers beneath it, and the story fails when text falls
 * below 4.5:1 or a mark below 3:1. A control gaining a colour cannot reach the
 * window without passing here, and a disabled control, which WCAG 1.4.3 leaves
 * out, is measured and reported but checked against no ratio.
 *
 * Each reading is printed as a `CONTRAST|kind|name|ratio` line.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import { Plus } from '@lucide/vue'
import { BarChip } from '../bar-chip'
import type { BarChipTone } from '../bar-chip'
import { Button } from '../button'
import type { ButtonVariants } from '../button'
import { Menu } from '../menu'
import { SegmentedControl } from '../segmented'
import { Select } from '../select'
import { Switch } from '../switch'
import { hoverOver } from '@/shared/fixtures/colour'
import { DARK, expectDark } from '@/shared/fixtures/theme'
import {
  expectFill,
  getFillRatio,
  expectText,
  finishMotion,
  getTextRatio,
  note,
} from '../../../../.storybook/contrastBrowser'
import { LARGE_RATIO } from '../../../../.storybook/contrast'

type Variant = NonNullable<ButtonVariants['variant']>
type Size = NonNullable<ButtonVariants['size']>

const VARIANTS: readonly Variant[] = ['solid', 'outline', 'danger', 'ghost']
const SIZES: readonly Size[] = ['default', 'small', 'icon', 'icon-small']
const GROUNDS = {
  window: 'var(--numen-surface)',
  card: 'var(--numen-raised)',
  panel: 'var(--numen-panel-bg)',
} as const

const isIconOnly = (size: Size): boolean => size === 'icon' || size === 'icon-small'

const meta: Meta = {
  title: 'Contrast/Controls',
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Every control that puts text or a mark on a solid ground, measured as WCAG 2 ' +
          'measures it, in the light set and the dark one.',
      },
    },
  },
}

export default meta
type Story = StoryObj

const frame = (body: string) =>
  `<div class="numen" style="padding:16px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">${body}</div>`

/* --------------------------------- Buttons --------------------------------- */

const buttons = () => ({
  components: { Button, Plus },
  setup: () => ({ VARIANTS, SIZES, GROUNDS, isIconOnly }),
  template: frame(`
    <div v-for="(ground, groundName) in GROUNDS" :key="groundName"
      style="display:flex;flex-wrap:wrap;gap:8px;padding:8px;align-items:center"
      :style="{ background: ground }">
      <template v-for="variant in VARIANTS" :key="variant">
        <template v-for="size in SIZES" :key="size">
          <template v-for="state in ['rest', 'hover', 'focus', 'disabled']" :key="state">
            <Button :variant="variant" :size="size" :disabled="state === 'disabled'"
              :data-case="[groundName, variant, size, state, 'plain'].join('|')"
              :aria-label="isIconOnly(size) ? 'Add' : undefined">
              <Plus v-if="isIconOnly(size)" width="14" height="14" />
              <template v-else>Done</template>
            </Button>
            <Button v-if="!isIconOnly(size)" :variant="variant" :size="size" :disabled="state === 'disabled'"
              :data-case="[groundName, variant, size, state, 'with icon'].join('|')">
              <Plus width="14" height="14" />Add
            </Button>
          </template>
        </template>
      </template>
    </div>`),
})

async function putInState(element: HTMLElement, state: string): Promise<void> {
  if (state === 'hover') await hoverOver(element)
  if (state === 'focus') element.focus()
  finishMotion(element)
}

async function playButtons(canvasElement: HTMLElement): Promise<void> {
  const cases = Array.from(canvasElement.querySelectorAll<HTMLElement>('[data-case]'))
  // Per variant and ground: two text sizes, each plain and with an icon, and two icon sizes,
  // each in four states.
  await expect(cases.length).toBe(3 * VARIANTS.length * (2 * 2 * 4 + 2 * 4))
  for (const element of cases) {
    const [ground = '', variant = '', size = '', state = '', kind = ''] = (
      element.dataset['case'] ?? ''
    ).split('|')
    const name = `${variant} ${size} ${kind} on ${ground}, ${state}`
    await putInState(element, state)
    if (state === 'disabled') {
      note('button', name, getTextRatio(element))
    } else if (isIconOnly(size as Size)) {
      await expectText(element, 'button', name, LARGE_RATIO)
    } else {
      await expectText(element, 'button', name)
    }
    if (state === 'hover') await userEvent.unhover(element)
    if (state === 'focus') element.blur()
  }
}

export const ButtonsLight: Story = {
  render: buttons,
  play: async ({ canvasElement }) => playButtons(canvasElement),
}

export const ButtonsDark: Story = {
  render: buttons,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playButtons(canvasElement)
  },
}

/* ----------------------------- Segmented control ---------------------------- */

const segmented = () => ({
  components: { SegmentedControl },
  setup: () => ({ model: ref('b'), off: ref('') }),
  template: frame(`
    <div style="display:flex;gap:16px;flex-wrap:wrap">
      <SegmentedControl v-model="model" data-case="enabled"
        :choices="[{ id: 'a', text: 'Light' }, { id: 'b', text: 'Dark' }, { id: 'c', text: 'System' }]" />
      <SegmentedControl v-model="model" is-disabled data-case="disabled"
        :choices="[{ id: 'a', text: 'Light' }, { id: 'b', text: 'Dark' }]" />
    </div>`),
})

async function playSegmented(canvasElement: HTMLElement): Promise<void> {
  for (const group of canvasElement.querySelectorAll<HTMLElement>('[data-case]')) {
    const isDisabled = group.dataset['case'] === 'disabled'
    for (const item of group.querySelectorAll<HTMLElement>('[role="radio"]')) {
      const isChosen = item.getAttribute('data-state') === 'checked'
      const name = `${group.dataset['case']} ${item.textContent?.trim()} ${isChosen ? 'chosen' : 'not chosen'}`
      for (const state of ['rest', 'hover', 'focus']) {
        if (isDisabled) {
          note('segmented', `${name}, ${state}`, getTextRatio(item))
          break
        }
        await putInState(item, state)
        await expectText(item, 'segmented', `${name}, ${state}`)
        if (state === 'hover') await userEvent.unhover(item)
        if (state === 'focus') item.blur()
      }
    }
  }
}

export const SegmentedLight: Story = {
  render: segmented,
  play: async ({ canvasElement }) => playSegmented(canvasElement),
}

export const SegmentedDark: Story = {
  render: segmented,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playSegmented(canvasElement)
  },
}

/* ---------------------------------- Switch --------------------------------- */

const switches = () => ({
  components: { Switch },
  setup: () => ({ on: ref(true), off: ref(false) }),
  template: frame(`
    <div style="display:flex;gap:16px;align-items:center">
      <Switch v-model="on" data-case="on" aria-label="On" />
      <Switch v-model="off" data-case="off" aria-label="Off" />
      <Switch v-model="on" is-disabled data-case="on disabled" aria-label="On, disabled" />
      <Switch v-model="off" is-disabled data-case="off disabled" aria-label="Off, disabled" />
    </div>`),
})

async function playSwitches(canvasElement: HTMLElement): Promise<void> {
  for (const track of canvasElement.querySelectorAll<HTMLElement>('[data-case]')) {
    const name = track.dataset['case'] ?? ''
    const thumb = track.firstElementChild as HTMLElement
    if (name.includes('disabled')) {
      note('switch thumb', name, getFillRatio(thumb))
      continue
    }
    await expectFill(thumb, 'switch thumb', name)
    // The track against the window is the other half of telling it on from off.
    await expectFill(track, 'switch track', name)
  }
}

export const SwitchLight: Story = {
  render: switches,
  play: async ({ canvasElement }) => playSwitches(canvasElement),
}

export const SwitchDark: Story = {
  render: switches,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playSwitches(canvasElement)
  },
}

/* ------------------------------ Menu and Select ----------------------------- */

const ITEMS = [
  { id: 'open', text: 'Open', detail: 'In this tab' },
  { id: 'child', text: 'New child note' },
  { id: 'gone', text: 'Delete', isDisabled: true },
]

const menus = () => ({
  components: { Menu, Select },
  setup: () => ({ ITEMS, chosen: ref('small') }),
  template: frame(`
    <div style="display:flex;gap:16px;align-items:start">
      <Select v-model="chosen" name="Size" data-case="select"
        :choices="[{ id: 'small', text: 'Small' }, { id: 'large', text: 'Large' }]" />
      <Menu :items="ITEMS" :at="{ x: 220, y: 80 }" is-open trigger="keyboard" current="open"
        name="Actions" :margin="8" />
    </div>`),
})

async function playMenus(canvasElement: HTMLElement): Promise<void> {
  const trigger = canvasElement.querySelector<HTMLElement>('[data-slot="select"]')
  if (!trigger) throw new Error('the select is not drawn')
  await expectText(trigger.querySelector('.select__reading') as HTMLElement, 'select', 'trigger')
  await putInState(trigger, 'hover')
  await expectText(
    trigger.querySelector('.select__reading') as HTMLElement,
    'select',
    'trigger, hover',
  )
  await userEvent.unhover(trigger)

  await waitFor(() =>
    expect(document.body.querySelectorAll('.menu__item').length).toBeGreaterThan(0),
  )
  const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
  for (const row of rows) {
    const name = row.querySelector('.menu__text')?.textContent ?? 'row'
    const text = row.querySelector('.menu__text') as HTMLElement
    if (row.hasAttribute('disabled')) {
      note('menu row', `${name}, disabled`, getTextRatio(text))
      continue
    }
    await expectText(text, 'menu row', `${name}, rest`)
    await hoverOver(row)
    finishMotion(row)
    await expectText(text, 'menu row', `${name}, highlighted by the pointer`)
    const detail = row.querySelector<HTMLElement>('.menu__detail')
    if (detail) await expectText(detail, 'menu row detail', `${name}, highlighted`)
    await userEvent.unhover(row)
    row.focus()
    finishMotion(row)
    await expectText(text, 'menu row', `${name}, focused`)
    row.blur()
  }
}

export const MenuLight: Story = {
  render: menus,
  play: async () => playMenus(document.body),
}

export const MenuDark: Story = {
  render: menus,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playMenus(canvasElement)
  },
}

/* --------------------------------- Bar chip -------------------------------- */

const CHIP_TONES: readonly BarChipTone[] = ['neutral', 'accent', 'alarm']

const barChips = () => ({
  components: { BarChip },
  setup: () => ({ CHIP_TONES }),
  template: frame(`
    <div style="display:flex;gap:16px;align-items:center;padding:8px;background:var(--numen-titlebar)">
      <template v-for="tone in CHIP_TONES" :key="tone">
        <BarChip v-for="state in ['rest', 'hover']" :key="state" :tone="tone"
          :label="[tone, state].join('|')">3 of 7</BarChip>
      </template>
    </div>
    <div style="display:flex;gap:96px;align-items:center;padding:8px;background:var(--numen-titlebar)">
      <BarChip v-for="tone in CHIP_TONES" :key="tone" :tone="tone" :open="true"
        :label="[tone, 'open'].join('|')" popover-label="Tide details">
        3 of 7
        <template #popover><p style="margin:0;padding:8px">High water at 14:20</p></template>
      </BarChip>
    </div>`),
})

async function playBarChips(canvasElement: HTMLElement): Promise<void> {
  const chips = Array.from(canvasElement.querySelectorAll<HTMLElement>('button[aria-label*="|"]'))
  await expect(chips.length).toBe(CHIP_TONES.length * 3)
  for (const chip of chips) {
    const [tone = '', state = ''] = (chip.getAttribute('aria-label') ?? '').split('|')
    await putInState(chip, state)
    await expectText(chip, 'bar chip', `${tone}, ${state}`)
    if (state === 'hover') await userEvent.unhover(chip)
  }
}

export const BarChipLight: Story = {
  render: barChips,
  play: async ({ canvasElement }) => playBarChips(canvasElement),
}

export const BarChipDark: Story = {
  render: barChips,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playBarChips(canvasElement)
  },
}
