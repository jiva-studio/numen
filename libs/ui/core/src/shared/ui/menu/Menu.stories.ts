/**
 * Every situation the menu has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`, which is the only place the
 * things this component exists for can fail — being clipped by what it stands
 * inside, folding back at an edge, and holding the keyboard.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { nextTick, onMounted, ref, type Component } from 'vue'
import { Copy, CornerDownRight, FileText } from '@lucide/vue'
import Menu from './Menu.vue'
import { MENU_TRIGGERS_ALL, type MenuItem, type MenuTrigger } from './items/item'
import {
  ARABIC,
  DEVANAGARI,
  EMPTY,
  LINK,
  LONG,
  RUSSIAN,
  UNBREAKABLE,
} from '@/shared/fixtures/prose'

interface Knobs {
  items: readonly MenuItem[]
  /** What each item is drawn with, by identity. Nothing named draws nothing. */
  icons?: Readonly<Record<string, Component>>
  at: { x: number; y: number }
  trigger: MenuTrigger
  /** Which item is the one in force, if the menu names one. */
  current: string | null
  /** Whether the name of a group is drawn over it. */
  hasGroups: boolean
  margin: number
  name: string
  isSearchable?: boolean
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
const renderMenu = (args: Knobs) => ({
  components: { Menu },
  setup() {
    const open = ref(true)
    const pos = ref(args.at.x < 0 ? { x: window.innerWidth + args.at.x, y: args.at.y } : args.at)
    const node = ref<HTMLElement | null>(null)
    const from = ref<HTMLElement | null>(null)

    onMounted(() => (from.value = node.value))

    const ask = (event: MouseEvent) => {
      event.preventDefault()
      pos.value = { x: event.clientX, y: event.clientY }
      from.value = event.currentTarget as HTMLElement
      open.value = true
    }

    return { args, open, at: pos, node, from, ask }
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
        :is-open="open"
        :from="from"
        :trigger="args.trigger"
        :current="args.current"
        :has-groups="args.hasGroups"
        :margin="args.margin"
        :name="args.name"
        :is-searchable="args.isSearchable"
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
    trigger: { control: 'inline-radio', options: MENU_TRIGGERS_ALL },
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
    trigger: 'pointer',
    current: null,
    hasGroups: false,
    margin: 8,
    name: 'Menu',
    onChoose: fn(),
    onDismiss: fn(),
  },
  render: renderMenu,
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
    await userEvent.click(within(menuElement()!).getByRole('menuitem', { name: 'New child note' }))
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
  args: { trigger: 'keyboard' },
  play: async ({ canvasElement }) => {
    const node = within(canvasElement).getByRole('button', { name: 'A node' })
    const getItem = (name: string) => within(menuElement()!).getByRole('menuitem', { name })

    await expect(getItem('Open')).toHaveFocus()

    await userEvent.keyboard('{Tab}')
    await expect(getItem('New child note')).toHaveFocus()

    await userEvent.keyboard('{Escape}')
    await waitFor(async () => {
      await expect(menuElement()).toBeNull()
    })
    await expect(node).toHaveFocus()
  },
}

/**
 * Asked for inside a box that clips everything in it, at the corner getFurthest
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
      const pos = ref({ x: 0, y: 0 })
      const from = ref<HTMLElement | null>(null)
      const ask = (event: MouseEvent) => {
        event.preventDefault()
        pos.value = { x: event.clientX, y: event.clientY }
        from.value = event.currentTarget as HTMLElement
        open.value = true
      }
      return { args, open, at: pos, from, ask }
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
          <Menu :items="args.items" :at="at" :is-open="open" :from="from" :margin="args.margin" />
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

    await expect(box.height).toBeGreaterThan(clipping.getBoundingClientRect().height)
    await expect(clipping.contains(menu)).toBe(false)

    // Whole at its current position: what is under its far edge is the menu itself,
    // which is exactly what being clipped would take away.
    const underElement = document.elementFromPoint(box.left + box.width / 2, box.bottom - 4)
    await expect(menu.contains(underElement)).toBe(true)

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
  args: { trigger: 'keyboard', current: 'child' },
  play: async () => {
    const selected = within(menuElement()!).getAllByRole('menuitemradio')
    await expect(selected).toHaveLength(4)
    await expect(selected[1]).toHaveFocus()
    await expect(selected.map((one) => one.getAttribute('aria-checked'))).toEqual([
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

/** Far more than fits: the list scrolls and the menu still relies on screen. */
export const FarTooMany: Story = {
  args: {
    items: Array.from({ length: 60 }, (_, index) => ({
      id: `item-${index}`,
      text: `Something to do (${index + 1})`,
    })),
  },
  play: async () => {
    const menu = menuElement()!
    await expect(menu.getBoundingClientRect().bottom).toBeLessThanOrEqual(window.innerHeight)
    await expect(menu.getBoundingClientRect().height).toBeLessThanOrEqual(24 * 16)
    await expect(menu.scrollHeight).toBeGreaterThan(menu.clientHeight)
  },
}

/** The search line is a row of its own: the items scroll under it and never into it. */
export const SearchStaysAboveTheList: Story = {
  args: {
    isSearchable: true,
    hasGroups: true,
    items: Array.from({ length: 60 }, (_, index) => ({
      id: `item-${index}`,
      text: `Something to do (${index + 1})`,
      group: index < 30 ? 'First' : 'Second',
    })),
  },
  play: async () => {
    const menu = menuElement()!
    const bar = menu.querySelector<HTMLElement>('.menu__bar')!
    const first = menu.querySelector<HTMLElement>('.menu__group-name')!
    await expect(first.getBoundingClientRect().top).toBeGreaterThanOrEqual(
      bar.getBoundingClientRect().bottom,
    )
    menu.scrollTop = 200
    await expect(bar.getBoundingClientRect().top).toBeLessThanOrEqual(
      menu.getBoundingClientRect().top + 8,
    )
    await expect(menu.getBoundingClientRect().height).toBeLessThanOrEqual(24 * 16)
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
    trigger: 'keyboard',
    items: [
      { id: 'open', text: 'Open' },
      { id: 'ask', text: 'Ask the agent about this note', isDisabled: true },
      { id: 'copy', text: 'Copy path' },
    ],
  },
  play: async () => {
    // The keyboard passes over it in both directions.
    const getItem = (name: string) => within(menuElement()!).getByRole('menuitem', { name })

    await expect(getItem('Open')).toHaveFocus()
    await userEvent.keyboard('{ArrowDown}')
    await expect(getItem('Copy path')).toHaveFocus()
    await userEvent.keyboard('{ArrowUp}')
    await expect(getItem('Open')).toHaveFocus()
  },
}

/**
 * Groups, ruled where one gives way to the next. What the groups mean is the
 * caller's; the menu draws a line where the word changes and nothing else.
 */
export const Grouped: Story = {
  args: {
    items: [
      { id: 'open', text: 'Open note', group: 'open' },
      { id: 'travel', text: 'Show in plex', group: 'open' },
      { id: 'note', text: 'New note', group: 'file' },
      { id: 'folder', text: 'New folder', group: 'file' },
      { id: 'rename', text: 'Rename', group: 'file' },
      { id: 'child', text: 'New child note', group: 'plex' },
      { id: 'title', text: 'Change title', group: 'plex' },
      { id: 'remove', text: 'Remove note', group: 'gone' },
    ],
  },
  play: async () => {
    const menu = within(menuElement()!)
    // that carry on a group.
    await expect(menu.getAllByRole('separator')).toHaveLength(3)
    // The keyboard passes over the rules: they are drawn, not chosen.
    await userEvent.keyboard('{ArrowDown}')
    await expect(menu.getByRole('menuitem', { name: 'Open note' })).toHaveFocus()
    await userEvent.keyboard('{ArrowUp}')
    await expect(menu.getByRole('menuitem', { name: 'Remove note' })).toHaveFocus()
  },
}

/** Groups named over the run they open, which is what a list of choices does. */
export const GroupsNamed: Story = {
  args: {
    items: [
      { id: 'numen', text: 'numen', group: 'Built-in' },
      { id: 'paper', text: 'paper', group: 'Built-in' },
      { id: 'sea', text: 'sea', group: 'Yours' },
    ],
    current: 'paper',
    hasGroups: true,
  },
  play: async () => {
    const menu = menuElement()!
    const named = Array.from(menu.querySelectorAll('.menu__group-name')).map((one) =>
      one.textContent?.trim(),
    )
    await expect(named).toEqual(['Built-in', 'Yours'])
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

/**
 * A menu opened near the bottom of a scrolling container, which scrolls
 * itself as the menu opens. The menu stays open through that scroll, and a
 * scroll the person makes afterwards closes it.
 */
const renderScrollingOpening = (args: Knobs) => ({
  components: { Menu },
  setup() {
    const open = ref(false)
    const pos = ref({ x: 0, y: 0 })
    const from = ref<HTMLElement | null>(null)
    const scroller = ref<HTMLElement | null>(null)

    const ask = async (event: MouseEvent) => {
      from.value = event.currentTarget as HTMLElement
      const box = from.value.getBoundingClientRect()
      pos.value = { x: box.left, y: box.bottom }
      open.value = true
      await nextTick()
      scroller.value?.scrollTo({ top: scroller.value.scrollHeight })
    }

    return { args, open, at: pos, from, scroller, ask }
  },
  template: `
    <div
      ref="scroller"
      data-scroller
      class="numen"
      style="height:100vh;overflow-y:auto;background:var(--numen-surface)"
    >
      <div style="height:150vh"></div>
      <button
        type="button"
        style="margin-inline-start:40px;padding:10px 18px;border-radius:6px;border:1px solid var(--numen-rule);background:var(--numen-raised);color:var(--numen-ink);font-family:var(--numen-font-sans)"
        @click="ask"
      >Language</button>
      <div style="height:40vh"></div>
      <Menu
        :items="args.items"
        :at="at"
        :is-open="open"
        :from="from"
        @dismiss="open = false; args.onDismiss()"
      />
    </div>
  `,
})

const scrollerOf = (canvasElement: HTMLElement): HTMLElement => {
  const scroller = canvasElement.querySelector<HTMLElement>('[data-scroller]')
  if (!scroller) throw new Error('The story draws no scrolling container')
  return scroller
}

export const OpenedWhileTheContainerScrolls: Story = {
  render: renderScrollingOpening,
  play: async ({ args, canvasElement }) => {
    const scroller = scrollerOf(canvasElement)
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Language' }))
    await waitFor(async () => {
      await expect(scroller.scrollTop).toBeGreaterThan(0)
    })
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))

    await expect(menuElement()).not.toBeNull()
    await expect(args.onDismiss).not.toHaveBeenCalled()
  },
}

export const ClosedByAScrollOfThePerson: Story = {
  render: renderScrollingOpening,
  play: async ({ args, canvasElement }) => {
    const scroller = scrollerOf(canvasElement)
    await userEvent.click(within(canvasElement).getByRole('button', { name: 'Language' }))
    await waitFor(async () => {
      await expect(scroller.scrollTop).toBeGreaterThan(0)
    })
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))
    await expect(menuElement()).not.toBeNull()

    scroller.dispatchEvent(new WheelEvent('wheel', { deltaY: -120, bubbles: true }))
    scroller.scrollTo({ top: 0 })

    await waitFor(async () => {
      await expect(args.onDismiss).toHaveBeenCalled()
    })
    await waitFor(async () => {
      await expect(menuElement()).toBeNull()
    })
  },
}
