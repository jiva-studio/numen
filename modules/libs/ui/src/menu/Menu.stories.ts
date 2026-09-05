/**
 * Every situation the menu has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`, which is the only place the
 * things this component exists for can fail — being clipped by what it stands
 * inside, folding back at an edge, and holding the keyboard.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { onMounted, ref, type Component } from 'vue'
import { Copy, CornerDownRight, FileText } from '@lucide/vue'
import Menu from './Menu.vue'
import { MENU_OPENINGS_ALL, type MenuItem, type MenuOpening } from './item'
import { ARABIC, DEVANAGARI, EMPTY, LINK, LONG, RUSSIAN, UNBREAKABLE } from '@/fixtures/prose'

interface Knobs {
  items: readonly MenuItem[]
  /** What each item is drawn with, by identity. Nothing named draws nothing. */
  icons?: Readonly<Record<string, Component>>
  at: { x: number; y: number }
  opening: MenuOpening
  /** Which item is the one in force, if the menu names one. */
  current: string | null
  /** Whether the name of a band is drawn over it. */
  bands: boolean
  margin: number
  name: string
  onChoose: (id: string) => void
  onDismiss: () => void
}

/** What the plan asks a node for. Nothing here knows what any of them do. */
const ITEMS: MenuItem[] = [
  { id: 'open', text: 'Open' },
  { id: 'child', text: 'New child note' },
  { id: 'ask', text: 'Ask the agent about this note' },
  { id: 'copy', text: 'Copy path' },
]

/** The menu's own list, so the placement is read off the element it decorates. */
const menuElement = () => document.body.querySelector<HTMLElement>('.menu')

/**
 * A surface with something on it to ask a menu of. The menu opens where it was
 * asked, and the story plays the caller's part: it holds whether the menu is
 * open, and it puts it away when the menu says so.
 */
const asked = (args: Knobs) => ({
  components: { Menu },
  setup() {
    const open = ref(true)
    const at = ref(
      args.at.x < 0 ? { x: window.innerWidth + args.at.x, y: args.at.y } : args.at,
    )
    const node = ref<HTMLElement | null>(null)
    const from = ref<HTMLElement | null>(null)

    onMounted(() => (from.value = node.value))

    const ask = (event: MouseEvent) => {
      event.preventDefault()
      at.value = { x: event.clientX, y: event.clientY }
      from.value = event.currentTarget as HTMLElement
      open.value = true
    }

    return { args, open, at, node, from, ask }
  },
  template: `
    <div class="numen" style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
      <button
        ref="node"
        type="button"
        style="padding:10px 18px;border-radius:6px;border:1px solid var(--numen-rule);background:var(--numen-raised);color:var(--numen-ink);font-family:var(--numen-font-sans)"
        @contextmenu="ask"
      >A node</button>
      <Menu
        :items="args.items"
        :at="at"
        :open="open"
        :from="from"
        :opening="args.opening"
        :current="args.current"
        :bands="args.bands"
        :margin="args.margin"
        :name="args.name"
        @choose="args.onChoose"
        @dismiss="open = false; args.onDismiss()"
      >
        <template v-if="args.icons" #icon="{ id }">
          <component
            :is="args.icons[id]"
            v-if="args.icons[id]"
            style="inline-size:100%;block-size:100%;stroke-width:1.875"
          />
        </template>
      </Menu>
    </div>
  `,
})

const meta = {
  title: 'Controls/Menu',
  component: Menu,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'A list of things that can be chosen, put where it was asked for. ' +
          'It is drawn at the end of the document, so nothing it stands ' +
          'inside can clip it, and it is placed against the area it is drawn ' +
          'into. It takes items and a point and says which item was chosen — ' +
          'what the items are and what choosing one does are the caller’s.',
      },
    },
  },
  argTypes: {
    opening: { control: 'inline-radio', options: MENU_OPENINGS_ALL },
    current: { control: 'text' },
    margin: { control: { type: 'range', min: 0, max: 48, step: 2 } },
    name: { control: 'text' },
    items: { table: { disable: true } },
    icons: { table: { disable: true } },
    at: { table: { disable: true } },
    onChoose: { table: { disable: true } },
    onDismiss: { table: { disable: true } },
  },
  args: {
    items: ITEMS,
    at: { x: 480, y: 300 },
    opening: 'pointer',
    current: null,
    bands: false,
    margin: 8,
    name: 'Menu',
    onChoose: fn(),
    onDismiss: fn(),
  },
  render: asked,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

/** Every setting, live, and a menu to ask for by right-clicking the node. */
export const Playground: Story = {}

/**
 * Choosing one. The identifier is handed back as given, and the menu asks to
 * be put away in the same breath.
 */
export const Choosing: Story = {
  play: async ({ args }) => {
    await userEvent.click(
      within(menuElement()!).getByRole('menuitem', { name: 'New child note' }),
    )
    await expect(args.onChoose).toHaveBeenCalledWith('child')
    await waitFor(async () => {
      await expect(menuElement()).toBeNull()
    })
  },
}

/**
 * Opened by hand: nothing is chosen, so Enter chooses nothing, and the first
 * step down lands on the first item.
 *
 * Only a browser can answer it. Enter reaches whatever holds the keyboard, so
 * what is lit and what would be chosen are one fact.
 */
export const OpenedByHand: Story = {
  play: async ({ args }) => {
    const menu = menuElement()!
    await expect(menu).toHaveFocus()
    await expect(menu.querySelector('[role="menuitem"]:focus')).toBeNull()

    await userEvent.keyboard('{Enter}')
    await expect(args.onChoose).not.toHaveBeenCalled()

    await userEvent.keyboard('{ArrowDown}')
    await expect(within(menu).getByRole('menuitem', { name: 'Open' })).toHaveFocus()

    await userEvent.keyboard('{Enter}')
    await expect(args.onChoose).toHaveBeenCalledWith('open')
  },
}

/**
 * Escape, and the keyboard goes back to the thing the menu was asked of.
 *
 * Only a browser can answer it: whether the keyboard actually lands on the
 * first item of a list drawn at the far end of the document, and whether it
 * finds its way back to an element the menu never contained.
 */
export const GivingItBack: Story = {
  args: { opening: 'keyboard' },
  play: async ({ canvasElement }) => {
    const node = within(canvasElement).getByRole('button', { name: 'A node' })
    const named = (name: string) => within(menuElement()!).getByRole('menuitem', { name })

    await expect(named('Open')).toHaveFocus()

    await userEvent.keyboard('{Tab}')
    await expect(named('New child note')).toHaveFocus()

    await userEvent.keyboard('{Escape}')
    await waitFor(async () => {
      await expect(menuElement()).toBeNull()
    })
    await expect(node).toHaveFocus()
  },
}

/**
 * Asked for inside a box that clips everything in it, at the corner furthest
 * from where a menu would like to open.
 *
 * This is the whole reason the menu is drawn where it is drawn. In jsdom
 * nothing is laid out and nothing is clipped, so this can only fail here.
 */
export const NotClipped: Story = {
  render: (args) => ({
    components: { Menu },
    setup() {
      const open = ref(false)
      const at = ref({ x: 0, y: 0 })
      const from = ref<HTMLElement | null>(null)
      const ask = (event: MouseEvent) => {
        event.preventDefault()
        at.value = { x: event.clientX, y: event.clientY }
        from.value = event.currentTarget as HTMLElement
        open.value = true
      }
      return { args, open, at, from, ask }
    },
    template: `
      <div class="numen" style="height:100vh;display:grid;place-items:center;background:var(--numen-surface)">
        <div
          data-clipping
          style="width:200px;height:110px;overflow:hidden;position:relative;outline:1px solid var(--numen-rule)"
        >
          <button
            type="button"
            style="position:absolute;inset-block-end:6px;inset-inline-end:6px;padding:8px 14px;border-radius:6px;border:1px solid var(--numen-rule);background:var(--numen-raised);color:var(--numen-ink);font-family:var(--numen-font-sans)"
            @contextmenu="ask"
          >A node at the corner</button>

          <!-- Written where a menu would be written: inside the thing that
               asked for it, which is the thing that clips. -->
          <Menu :items="args.items" :at="at" :open="open" :from="from" :margin="args.margin" />
        </div>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const node = within(canvasElement).getByRole('button', { name: 'A node at the corner' })
    await userEvent.pointer({ keys: '[MouseRight]', target: node })

    await waitFor(async () => {
      await expect(menuElement()).not.toBeNull()
    })
    const menu = menuElement()!
    const box = menu.getBoundingClientRect()
    const clipping = canvasElement.querySelector('[data-clipping]')!

    // Taller than the box it was asked from, and not standing inside it.
    await expect(box.height).toBeGreaterThan(clipping.getBoundingClientRect().height)
    await expect(clipping.contains(menu)).toBe(false)

    // Whole where it stands: what is under its far edge is the menu itself,
    // which is exactly what being clipped would take away.
    const under = document.elementFromPoint(box.left + box.width / 2, box.bottom - 4)
    await expect(menu.contains(under)).toBe(true)

    // And inside the area it is placed in, on every side.
    await expect(box.left).toBeGreaterThanOrEqual(0)
    await expect(box.top).toBeGreaterThanOrEqual(0)
    await expect(box.right).toBeLessThanOrEqual(window.innerWidth)
    await expect(box.bottom).toBeLessThanOrEqual(window.innerHeight)
  },
}

/**
 * A choice between the items, one of them in force. Each says whether it is
 * the one, and the keyboard opens on it.
 */
export const TheOneInForce: Story = {
  args: { opening: 'keyboard', current: 'child' },
  play: async () => {
    const chosen = within(menuElement()!).getAllByRole('menuitemradio')
    await expect(chosen).toHaveLength(4)
    await expect(chosen[1]).toHaveFocus()
    await expect(chosen.map((one) => one.getAttribute('aria-checked'))).toEqual([
      'false',
      'true',
      'false',
      'false',
    ])
  },
}

/** Nothing to choose at all. */
export const Empty: Story = {
  args: { items: [] },
}

/** One thing to choose. */
export const One: Story = {
  args: { items: [{ id: 'open', text: 'Open' }] },
}

/**
 * Icons, which are the caller's. The room for one is kept on every item, so
 * the words line up down the menu whether or not each of them draws anything.
 */
export const Icons: Story = {
  args: {
    items: [...ITEMS, { id: 'nothing', text: 'Drawn as nothing' }],
    icons: { open: FileText, child: CornerDownRight, copy: Copy },
  },
}

/** Far more than fits: the list scrolls and the menu still stands on screen. */
export const FarTooMany: Story = {
  args: {
    items: Array.from({ length: 60 }, (_, at) => ({
      id: `item-${at}`,
      text: `Something to do (${at + 1})`,
    })),
  },
  play: async () => {
    const menu = menuElement()!
    await expect(menu.getBoundingClientRect().bottom).toBeLessThanOrEqual(window.innerHeight)
    await expect(menu.scrollHeight).toBeGreaterThan(menu.clientHeight)
  },
}

/** Not Latin, and not all in one direction. */
export const NotLatin: Story = {
  args: {
    items: [
      { id: 'devanagari', text: DEVANAGARI },
      { id: 'arabic', text: ARABIC },
      { id: 'russian', text: RUSSIAN },
    ],
  },
}

/** Far past any width a menu is drawn at. */
export const FarTooLong: Story = {
  args: {
    items: [
      { id: 'long', text: LONG },
      { id: 'open', text: 'Open' },
    ],
  },
}

/**
 * Asked for beside the far edge, where the room left is narrower than the
 * choices. A menu is as wide as the longest thing it offers, so the words are
 * read whole and the menu is the thing that moves.
 */
export const AskedForAtTheEdge: Story = {
  args: {
    // Counted back from the far edge: the room left is narrower than the
    // words, whatever the page is drawn at.
    at: { x: -200, y: 200 },
    items: [
      { id: 'wide', text: 'As wide as the longest of its words' },
      { id: 'near', text: 'Beside the edge it was asked for at' },
    ],
  },
  play: async ({ canvasElement }) => {
    const menu = document.body.querySelector<HTMLElement>('.menu')
    expect(menu).not.toBeNull()
    const lines = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__text'))
    expect(lines).toHaveLength(2)
    for (const line of lines) {
      // Nothing is cut short: what the line holds fits the room it has.
      expect(line.scrollWidth).toBeLessThanOrEqual(line.clientWidth)
    }
    // And it stands inside the page it was asked for from.
    const box = menu!.getBoundingClientRect()
    expect(box.right).toBeLessThanOrEqual(canvasElement.ownerDocument.documentElement.clientWidth)
    expect(box.left).toBeGreaterThanOrEqual(0)
  },
}

/** Nothing to break at: one word, and the URL anybody actually pastes. */
export const NothingToBreakAt: Story = {
  args: {
    items: [
      { id: 'word', text: UNBREAKABLE },
      { id: 'link', text: LINK },
    ],
  },
}

/** No text at all, on an item that is still there and still choosable. */
export const NoTextAtAll: Story = {
  args: {
    items: [
      { id: 'nothing', text: EMPTY },
      { id: 'open', text: 'Open' },
    ],
  },
}

/** An item that is drawn and announced, and cannot be chosen. */
export const NotChoosable: Story = {
  args: {
    opening: 'keyboard',
    items: [
      { id: 'open', text: 'Open' },
      { id: 'ask', text: 'Ask the agent about this note', disabled: true },
      { id: 'copy', text: 'Copy path' },
    ],
  },
  play: async () => {
    // The keyboard passes over it in both directions.
    const named = (name: string) => within(menuElement()!).getByRole('menuitem', { name })

    await expect(named('Open')).toHaveFocus()
    await userEvent.keyboard('{ArrowDown}')
    await expect(named('Copy path')).toHaveFocus()
    await userEvent.keyboard('{ArrowUp}')
    await expect(named('Open')).toHaveFocus()
  },
}

/**
 * Bands, ruled where one gives way to the next. What the bands mean is the
 * caller's; the menu draws a line where the word changes and nothing else.
 */
export const Banded: Story = {
  args: {
    items: [
      { id: 'open', text: 'Open the note', band: 'open' },
      { id: 'travel', text: 'Show in plex', band: 'open' },
      { id: 'note', text: 'New note', band: 'file' },
      { id: 'folder', text: 'New folder', band: 'file' },
      { id: 'rename', text: 'Rename', band: 'file' },
      { id: 'child', text: 'New child note', band: 'plex' },
      { id: 'title', text: 'Change title', band: 'plex' },
      { id: 'remove', text: 'Remove note', band: 'gone' },
    ],
  },
  play: async () => {
    const menu = within(menuElement()!)
    // A rule stands at each of the three joins, and above none of the items
    // that carry on a band.
    await expect(menu.getAllByRole('separator')).toHaveLength(3)
    // The keyboard passes over the rules: they are drawn, not chosen.
    await userEvent.keyboard('{ArrowDown}')
    await expect(menu.getByRole('menuitem', { name: 'Open the note' })).toHaveFocus()
    await userEvent.keyboard('{ArrowUp}')
    await expect(menu.getByRole('menuitem', { name: 'Remove note' })).toHaveFocus()
  },
}

/** Bands named over the run they open, which is what a list of choices does. */
export const BandsNamed: Story = {
  args: {
    items: [
      { id: 'numen', text: 'numen', band: 'Ships with numen' },
      { id: 'paper', text: 'paper', band: 'Ships with numen' },
      { id: 'sea', text: 'sea', band: 'Yours' },
    ],
    current: 'paper',
    bands: true,
  },
  play: async () => {
    const menu = menuElement()!
    const named = Array.from(menu.querySelectorAll('.menu__band')).map((one) =>
      one.textContent?.trim(),
    )
    await expect(named).toEqual(['Ships with numen', 'Yours'])
    // The names are drawn, not ruled: a name and a rule do the one job.
    await expect(menu.querySelectorAll('.menu__rule')).toHaveLength(0)
  },
}

/** What an item says beside its words: the address a model is fetched from. */
export const ASecondLine: Story = {
  args: {
    items: [
      { id: 'small', text: 'PP-OCRv6, small', detail: 'somewhere/PP-OCRv6_rec_tiny.onnx' },
      { id: 'large', text: 'PP-OCRv6, large' },
    ],
  },
  play: async () => {
    const menu = menuElement()!
    await expect(menu.querySelectorAll('.menu__detail')).toHaveLength(1)
  },
}

/** Typing lands the keyboard on the item the letters begin. */
export const TypingToJump: Story = {
  play: async () => {
    const menu = within(menuElement()!)
    await userEvent.keyboard('c')
    await expect(menu.getByRole('menuitem', { name: 'Copy path' })).toHaveFocus()
  },
}
