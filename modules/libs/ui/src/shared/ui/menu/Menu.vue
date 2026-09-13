<script setup lang="ts">
/**
 * A menu: a list of things that can be chosen, put where it was asked for.
 *
 * It is drawn at the end of the document, so nothing it stands inside can clip
 * it, and it is placed against the area it is drawn into. It takes items and a
 * point and says which item was chosen; what the items are and what choosing
 * one does are the caller's.
 */
import { computed, nextTick, onBeforeUnmount, useTemplateRef, watch } from 'vue'
import MenuRow from './MenuRow.vue'
import { groupItems, getLandingIndex, type MenuItem, type MenuOpening } from './item'
import { useMenuGround } from './ground'
import { useMenuKeys } from './keys'
import { useMenuPlacement } from './place'
import type { Position, Size } from '@/shared/lib/geometry'

const props = withDefaults(
  defineProps<{
    /** What can be chosen, in the order it is drawn. */
    items: readonly MenuItem[]
    /** Where it was asked for, in the coordinates of the area it is drawn into. */
    at: Position
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
     * Whether the name of a group is drawn over it. A menu whose groups are
     * named by identifiers draws none.
     */
    groups?: boolean
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
    groups: false,
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

/** The items with the rules that stand between their groups. */
const rows = computed(() => groupItems(props.items))

const { placed, setSize } = useMenuPlacement({
  at: () => props.at,
  viewport: () => props.viewport,
  margin: () => props.margin,
})

/** Its own size, which only the drawing knows. */
const measure = () => {
  const box = menu.value?.getBoundingClientRect()
  if (box) setSize({ width: box.width, height: box.height })
}

const { listen, release } = useMenuGround(menu, () => emit('dismiss'))

const { here, holdRow, goTo, onKey } = useMenuKeys(() => props.items, menu, () => Date.now())

const choose = (item: MenuItem) => {
  if (item.disabled) return
  emit('choose', item.id)
  emit('dismiss')
}

/** Whether the menu stands: what entering put in place, leaving takes away. */
let standing = false

const enter = async () => {
  if (standing) return
  standing = true
  listen()

  await nextTick()
  measure()
  goTo(getLandingIndex(props.opening, props.items, props.current))
}

const leave = () => {
  if (!standing) return
  standing = false
  release()
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

const menuStyle = computed(() => ({
  left: `${placed.value.x}px`,
  top: `${placed.value.y}px`,
  '--asking': `${props.asking}px`,
}))

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
      :style="menuStyle"
      @keydown="onKey"
    >
      <MenuRow
        v-for="(item, index) in rows"
        :ref="(row) => holdRow(item.id, row)"
        :key="item.id"
        :item="item"
        :current="current"
        :named="groups && Boolean(item.group) && (item.rule || index === 0)"
        :icons="Boolean($slots.icon)"
        @focus="here = index"
        @choose="choose(item)"
      >
        <slot name="icon" :id="item.id" />
      </MenuRow>

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
  /* A menu is as long as what it offers, and it scrolls only where the screen
     itself cannot hold it. */
  --tallest: calc(100vh - 1rem);
  --lift: var(--numen-lift-menu);

  position: fixed;
  z-index: var(--lift);
  /* As wide as the longest thing it offers, within these two widths — or as
     wide as what asked for it, where that is wider than either. */
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

.menu__silence {
  margin: 0;
}
</style>
