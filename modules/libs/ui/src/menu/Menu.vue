<script setup lang="ts">
/**
 * A menu: a list of things that can be chosen, put where it was asked for.
 *
 * It is drawn at the end of the document, so nothing it stands inside can clip
 * it, and it is placed against the area it is drawn into. It takes items and a
 * point and says which item was chosen; what the items are and what choosing
 * one does are the caller's.
 */
import { computed, nextTick, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { banded, landsOn, placeMenu, stepTo, type MenuItem, type MenuOpening } from './model'
import type { Point, Size } from '../lib/geometry'

const props = withDefaults(
  defineProps<{
    /** What can be chosen, in the order it is drawn. */
    items: readonly MenuItem[]
    /** Where it was asked for, in the coordinates of the area it is drawn into. */
    at: Point
    /** Whether it is drawn at all. */
    open?: boolean
    /** What opened it. Opened by hand it appears with nothing chosen. */
    opening?: MenuOpening
    /**
     * Which item is the one in force, by the caller's identifier. A menu
     * naming one offers a choice between its items: each says whether it is
     * the one, and the keyboard opens on it.
     */
    current?: string | null
    /** Where the keyboard goes back to once it closes. */
    from?: HTMLElement | SVGElement | null
    /** The area it is placed in. The browser's own by default. */
    viewport?: Size | null
    /**
     * Whether the name of a band is drawn over it. A menu whose bands are
     * named by identifiers draws none.
     */
    bands?: boolean
    /**
     * How wide what asked for it is. The menu is never narrower than that, and
     * grows past it for what it holds.
     */
    asking?: number
    /** Kept clear of that area's edges. */
    margin?: number
    /** Where it is drawn. The end of the document by default. */
    to?: string | HTMLElement
    /** What it is announced as. */
    name?: string
  }>(),
  {
    open: false,
    opening: 'pointer',
    current: null,
    from: null,
    viewport: null,
    bands: false,
    asking: 0,
    margin: 8,
    to: 'body',
    name: 'Menu',
  },
)

const emit = defineEmits<{
  /** An item was chosen. The identifier is the caller's, handed back as given. */
  (event: 'choose', id: string): void
  /** It asks to be put away. */
  (event: 'dismiss'): void
}>()

defineSlots<{
  /**
   * What is drawn before an item's words. The room for it is kept on every
   * item once the slot is filled, so the words line up down the menu whether
   * or not each of them draws anything.
   */
  icon(props: { id: string }): unknown
  /** What is said when there is nothing to choose. */
  silence(): unknown
}>()

const menu = useTemplateRef<HTMLElement>('menu')

/** Its own size, which only the drawing knows. Placement is worked out from it. */
const size = ref<Size>({ width: 0, height: 0 })

/** Which item the keyboard is on, or -1 when it is on none. */
const here = ref(-1)

/** The items with the rules that stand between their bands. */
const rows = computed(() => banded(props.items))

/** The area to stay inside. The browser's, unless a caller measures its own. */
const room = computed<Size>(
  () => props.viewport ?? { width: window.innerWidth, height: window.innerHeight },
)

const placed = computed(() =>
  placeMenu({
    at: props.at,
    size: size.value,
    viewport: room.value,
    margin: props.margin,
  }),
)

/** Each item as it is drawn, each under the item it stands for. */
const drawn = new Map<string, HTMLElement>()

const holdRow = (item: string, row: unknown): void => {
  if (row) drawn.set(item, row as HTMLElement)
  else drawn.delete(item)
}

const measure = () => {
  const element = menu.value
  if (!element) return
  const box = element.getBoundingClientRect()
  size.value = { width: box.width, height: box.height }
}

/** The keyboard onto an item, or onto the menu itself where there is none. */
const goTo = (index: number) => {
  here.value = index
  const item = props.items[index]
  const chosen = item ? drawn.get(item.id) : undefined
  if (chosen) chosen.focus()
  else menu.value?.focus()
}

const choose = (item: MenuItem) => {
  if (item.disabled) return
  emit('choose', item.id)
  emit('dismiss')
}

/**
 * A pointer, or a scroll, that did not happen inside the menu. A scroll of the
 * menu's own list is not the ground moving, and everything else is.
 */
const outside = (event: Event) => {
  const target = event.target
  if (target instanceof Node && menu.value?.contains(target)) return
  emit('dismiss')
}

const onWindowKey = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  event.preventDefault()
  emit('dismiss')
}

/** What has been typed to jump by, and when the last letter of it arrived. */
let typed = ''
let struck = 0

/** How long a run of letters stays one word. */
const TYPING = 1000

/**
 * The keyboard onto the next item beginning with what has been typed. A run of
 * letters is one word, and stands where it is while the word grows. One letter
 * struck again and again walks the items beginning with it.
 */
const jumpTo = (letter: string) => {
  const now = Date.now()
  typed = now - struck > TYPING ? letter : typed + letter
  struck = now
  const one = typed[0]!
  const drumming = [...typed].every((each) => each === one)
  const word = drumming ? one : typed
  const total = props.items.length
  const from = here.value < 0 ? 0 : here.value + (typed.length > 1 && !drumming ? 0 : 1)
  const said = word.toLowerCase()
  for (let step = 0; step < total; step += 1) {
    const at = (from + step) % total
    const item = props.items[at]
    if (!item || item.disabled) continue
    if (item.text.toLowerCase().startsWith(said)) {
      goTo(at)
      return
    }
  }
}

/** A key that stands for a letter a person meant to type. */
const letters = (event: KeyboardEvent): boolean =>
  event.key.length === 1 && event.key !== ' ' && !event.ctrlKey && !event.metaKey && !event.altKey

/**
 * The keyboard, while the menu is open. Tab moves within the items and wraps,
 * which is what keeps the keyboard inside a menu that stands over the page.
 */
const onKey = (event: KeyboardEvent) => {
  // Counting back from no item is counting back from the first.
  const step = (by: number, from = by < 0 ? Math.max(here.value, 0) : here.value) => {
    event.preventDefault()
    goTo(stepTo(props.items, from, by))
  }
  if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Tab') step(event.shiftKey ? -1 : 1)
  else if (letters(event)) {
    event.preventDefault()
    jumpTo(event.key)
  }
}

/** What the open menu installed on the window, if anything. */
let detach: (() => void) | null = null

const enter = async () => {
  if (detach) return
  window.addEventListener('pointerdown', outside, true)
  window.addEventListener('scroll', outside, true)
  window.addEventListener('resize', outside)
  window.addEventListener('keydown', onWindowKey)
  detach = () => {
    window.removeEventListener('pointerdown', outside, true)
    window.removeEventListener('scroll', outside, true)
    window.removeEventListener('resize', outside)
    window.removeEventListener('keydown', onWindowKey)
  }

  await nextTick()
  measure()
  goTo(landsOn(props.opening, props.items, props.current))
}

const leave = () => {
  if (!detach) return
  detach()
  detach = null
  here.value = -1
  const back = props.from
  if (back?.isConnected) back.focus()
}

watch(
  () => props.open,
  (now) => {
    if (now) void enter()
    else leave()
  },
  { immediate: true },
)

/** Measured again when what it holds changes, and when the point does. */
watch(
  () => [props.items, props.at],
  async () => {
    if (!props.open) return
    await nextTick()
    measure()
  },
)

// A menu can go while it is still open, and what it left on the window with it.
onBeforeUnmount(leave)
</script>

<template>
  <Teleport :to="to">
    <div
      v-if="open"
      ref="menu"
      class="menu numen panel-numen flex flex-col p-1.5 font-sans text-base text-ink"
      role="menu"
      tabindex="-1"
      :aria-label="name"
      :style="{
        left: `${placed.x}px`,
        top: `${placed.y}px`,
        '--asking': `${asking}px`,
      }"
      @keydown="onKey"
    >
      <template v-for="(item, index) in rows" :key="item.id">
        <p
          v-if="bands && item.band && (item.rule || index === 0)"
          class="menu__band px-2 py-1 text-hushed"
          aria-hidden="true"
        >
          {{ item.band }}
        </p>
        <hr v-else-if="item.rule" class="menu__rule" role="separator" />

        <button
          :ref="(row) => holdRow(item.id, row)"
          class="menu__item flex w-full items-center rounded-node px-2 py-1.5 text-left"
          type="button"
          :role="current === null ? 'menuitem' : 'menuitemradio'"
          :aria-checked="current === null ? undefined : item.id === current"
          tabindex="-1"
          :disabled="item.disabled"
          @focus="here = index"
          @click="choose(item)"
        >
          <span v-if="$slots.icon" class="menu__icon flex shrink-0 items-center">
            <slot name="icon" :id="item.id" />
          </span>
          <span class="menu__said flex min-w-0 flex-col">
            <span class="menu__text">{{ item.text }}</span>
            <span v-if="item.detail" class="menu__detail">{{ item.detail }}</span>
          </span>
        </button>
      </template>

      <p v-if="!items.length" class="menu__silence px-2 py-1.5 text-hushed">
        <slot name="silence">Nothing to do</slot>
      </p>
    </div>
  </Teleport>
</template>

<style scoped>
/* Placed by the two numbers the placement worked out, and standing over the
   page it was asked for from. */
.menu {
  /* How wide it may be, how much of the screen it takes before its list
     scrolls, and how far above the page it stands. The two widths are set in
     the interface's own units, so they grow with everything drawn beside them. */
  --narrowest: 11.25rem;
  --widest: 20rem;
  /* What asked for it, which it is never narrower than. */
  --asking: 0px;
  --tallest: 60vh;
  --lift: var(--numen-lift-menu);
  /* The room a rule keeps on each side of itself. */
  --parting: 0.25rem;
  /* How large an icon is drawn, and the room between it and the words. */
  --icon: 0.875rem;
  --icon-gap: 0.5rem;

  position: fixed;
  z-index: var(--lift);
  /* As wide as the longest thing it offers, and always between these two
     widths. */
  inline-size: max-content;
  min-inline-size: max(var(--narrowest), var(--asking));
  max-inline-size: max(var(--widest), var(--asking));
  max-block-size: var(--tallest);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.menu:focus-visible {
  outline: none;
}

/* One physical line, so it stays a hairline however large the interface is
   drawn. */
.menu__rule {
  block-size: 0;
  margin-block: var(--parting);
  border: 0;
  border-block-start: 1px solid var(--numen-panel-border);
}

.menu__item {
  cursor: default;
  user-select: none;
  -webkit-user-select: none;
}

.menu__item:hover:not(:disabled),
.menu__item:focus-visible {
  outline: none;
  background: var(--numen-bubble-bg);
}

.menu__item:disabled {
  color: var(--numen-edge-label);
}

/* The room an icon takes, kept whether or not the item draws one, so the words
   line up down the menu. What is drawn in it is the caller's. */
.menu__icon {
  inline-size: var(--icon);
  block-size: var(--icon);
  margin-inline-end: var(--icon-gap);
}

/* One line, then an ellipsis. A menu is read down its leading edge. */
.menu__text,
.menu__detail {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* What an item is beside its words: the address a model is fetched from, the
   place a file stands. */
.menu__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* The name of a band, set as this product sets a label over what it names. */
.menu__band {
  margin: 0;
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.menu__silence {
  margin: 0;
}
</style>
