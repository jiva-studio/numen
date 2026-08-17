<script setup lang="ts">
/**
 * A menu: a list of things that can be chosen, put where it was asked for.
 *
 * It is drawn at the end of the document, so nothing it stands inside can clip
 * it, and it is placed against the area it is drawn into rather than against
 * whatever asked for it. It takes items and a point and says which item was
 * chosen; what the items are and what choosing one does are the caller's.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { placeMenu, stepTo, type MenuItem } from './model'
import type { Point } from '../plex/model'
import type { Size } from '../plex/arrange'

const props = withDefaults(
  defineProps<{
    /** What can be chosen, in the order it is drawn. */
    items: readonly MenuItem[]
    /** Where it was asked for, in the coordinates of the area it is drawn into. */
    at: Point
    /** Whether it is drawn at all. */
    open?: boolean
    /** Where the keyboard goes back to once it closes. */
    from?: HTMLElement | SVGElement | null
    /** The area it is placed in. The browser's own by default. */
    viewport?: Size | null
    /** Kept clear of that area's edges. */
    margin?: number
    /** Where it is drawn. The end of the document by default. */
    to?: string | HTMLElement
    /** What it is announced as. */
    name?: string
  }>(),
  {
    open: false,
    from: null,
    viewport: null,
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
  /** What is said when there is nothing to choose. */
  silence(): unknown
}>()

const menu = ref<HTMLElement | null>(null)

/** Its own size, which only the drawing knows. Placement is worked out from it. */
const size = ref<Size>({ width: 0, height: 0 })

/** Which item the keyboard is on, or -1 when it is on none. */
const here = ref(-1)

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

/** In document order, so an index into the items is an index into these. */
const drawn = () =>
  Array.from(menu.value?.querySelectorAll<HTMLElement>('.menu__item') ?? [])

const measure = () => {
  const element = menu.value
  if (!element) return
  const box = element.getBoundingClientRect()
  size.value = { width: box.width, height: box.height }
}

const goTo = (index: number) => {
  here.value = index
  const chosen = drawn()[index]
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

/**
 * The keyboard, while the menu is open. Tab moves within the items and wraps,
 * which is what keeps the keyboard inside a menu that stands over the page.
 */
const onKey = (event: KeyboardEvent) => {
  const step = (by: number, from = here.value) => {
    event.preventDefault()
    goTo(stepTo(props.items, from, by))
  }
  if (event.key === 'ArrowDown') step(1)
  else if (event.key === 'ArrowUp') step(-1)
  else if (event.key === 'Home') step(1, -1)
  else if (event.key === 'End') step(-1, 0)
  else if (event.key === 'Tab') step(event.shiftKey ? -1 : 1)
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
  goTo(stepTo(props.items, -1, 1))
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

onMounted(() => {
  if (props.open) void enter()
})

// A menu can go while it is still open, and what it left on the window with it.
onBeforeUnmount(leave)
</script>

<template>
  <Teleport :to="to">
    <div
      v-if="open"
      ref="menu"
      class="menu numen flex flex-col rounded-panel border border-panel-rule bg-panel p-1.5 font-sans text-base text-ink shadow-panel backdrop-blur-panel"
      role="menu"
      tabindex="-1"
      :aria-label="name"
      :style="{ left: `${placed.x}px`, top: `${placed.y}px` }"
      @keydown="onKey"
    >
      <button
        v-for="(item, index) in items"
        :key="item.id"
        class="menu__item flex w-full items-center rounded-node px-2 py-1.5 text-left"
        type="button"
        role="menuitem"
        tabindex="-1"
        :disabled="item.disabled"
        @focus="here = index"
        @click="choose(item)"
      >
        <span class="menu__text min-w-0">{{ item.text }}</span>
      </button>

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
     scrolls, and how far above the page it stands. */
  --narrowest: 180px;
  --widest: 320px;
  --tallest: 60vh;
  --lift: 60;

  position: fixed;
  z-index: var(--lift);
  min-inline-size: var(--narrowest);
  max-inline-size: var(--widest);
  max-block-size: var(--tallest);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.menu:focus-visible {
  outline: none;
}

.menu__item {
  cursor: default;
  user-select: none;
}

.menu__item:hover:not(:disabled),
.menu__item:focus {
  outline: none;
  background: var(--numen-bubble-bg);
}

.menu__item:disabled {
  color: var(--numen-edge-label);
}

/* One line, then an ellipsis. A menu is read down its leading edge. */
.menu__text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu__silence {
  margin: 0;
}
</style>
