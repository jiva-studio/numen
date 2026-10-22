<script setup lang="ts">
/**
 * A menu: a list of things that can be chosen, put where it was asked for.
 *
 * it, and it is placed against the area it is drawn into. It takes items and a
 * point and says which item was chosen; what the items are and what choosing
 * one does are the caller's.
 */
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
import MenuRow from './MenuRow.vue'
import { filterItems, groupItems, getInitialIndex, type MenuItem, type MenuTrigger } from './item'
import { isLetter } from './typeahead'
import { useMenuDismissal } from './menuDismissal'
import { useMenuKeys } from './keys'
import { useMenuOpen } from './open'
import { useMenuPlacement } from './place'
import type { Position, Size } from '@/shared/lib/geometry'

const props = withDefaults(
  defineProps<{
    /** What can be chosen, in the order it is drawn. */
    items: readonly MenuItem[]
    /** Where it was asked for, in the coordinates of the area it is drawn into. */
    at: Position
    /** Whether it is drawn at all. */
    isOpen?: boolean
    /** What opened it. Opened by hand it appears with nothing chosen. */
    trigger?: MenuTrigger
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
    hasGroups?: boolean
    /**
     * How wide what asked for it is. The menu is never narrower than that, and
     * grows past it for what it holds.
     */
    asking?: number
    /** Kept clear of that area's edges. */
    margin?: number
    /** Where it is drawn. The end of the document by default. */
    to?: string | HTMLElement
    /** What the moment is. The window's clock by default. */
    clock?: () => number
    /** What it is announced as. */
    name?: string
    /**
     * Whether a line to search the items by stands above them. The items shown
     * are those that name every word typed in their text, detail or group.
     */
    searchable?: boolean
    /** What the search line is announced as. */
    searchName?: string
  }>(),
  {
    isOpen: false,
    trigger: 'pointer',
    current: null,
    from: null,
    viewport: null,
    hasGroups: false,
    asking: 0,
    margin: 8,
    to: 'body',
    name: 'Menu',
    searchable: false,
    searchName: 'Search',
    clock: () => Date.now(),
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

const search = useTemplateRef<HTMLInputElement>('search')
const query = ref('')

/** The items the search leaves, and all of them where there is none. */
const visibleItems = computed(() =>
  props.searchable ? filterItems(props.items, query.value) : props.items,
)

/** The items shown, with the rules that stand between their groups. */
const rows = computed(() => groupItems(visibleItems.value))

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

const { listen, release } = useMenuDismissal(menu, () => emit('dismiss'))

const { here, holdRow, goTo, onKey: onRowKey } = useMenuKeys(() => visibleItems.value, menu, props.clock)

const choose = (item: MenuItem) => {
  if (item.isDisabled) return
  emit('choose', item.id)
  emit('dismiss')
}

/**
 * A key pressed inside the menu. On the search line the arrows and Tab step
 * into the items and Enter chooses the first one; every other key is the line's
 * own. A letter struck on an item goes to the line.
 */
const onKey = (event: KeyboardEvent): void => {
  if (!props.searchable) return onRowKey(event)
  if (event.target === search.value) {
    if (event.key === 'Enter') {
      event.preventDefault()
      const first = visibleItems.value.find((one) => !one.isDisabled)
      if (first && query.value) choose(first)
    } else if (['ArrowDown', 'ArrowUp', 'Tab'].includes(event.key)) {
      here.value = -1
      onRowKey(event)
    }
    return
  }
  if (isLetter(event)) search.value?.focus()
  else onRowKey(event)
}

const isMenuVisible = computed(() => Boolean(props.isOpen))

useMenuOpen({
  isOpen: () => isMenuVisible.value,
  listen,
  release,
  measure,
  focusInitial,
  reset: () => {
    here.value = -1
    query.value = ''
  },
  getFrom: () => props.from,
})

/** The keyboard onto the search line where there is one, else onto the item in force. */
function focusInitial(): void {
  if (!props.searchable) {
    goTo(getInitialIndex(props.trigger, props.items, props.current))
    return
  }
  search.value?.focus()
  menu.value?.querySelector('[aria-checked="true"]')?.scrollIntoView?.({ block: 'nearest' })
}

/** The keyboard is on no item once the search changes. */
watch(query, () => (here.value = -1))

/** Measured again when what it holds changes, and when the point does. */
watch(
  () => [visibleItems.value, props.at],
  async () => {
    if (!isMenuVisible.value) return
    await nextTick()
    measure()
  },
)

const menuStyle = computed(() => ({
  left: `${placed.value.x}px`,
  top: `${placed.value.y}px`,
  '--asking': `${props.asking}px`,
  '--room': props.viewport
    ? `${Math.max(props.viewport.height - 2 * props.margin, 0)}px`
    : `calc(100dvh - ${2 * props.margin}px)`,
}))
</script>

<template>
  <Teleport :to="to">
    <div
      v-if="isMenuVisible"
      ref="menu"
      class="menu numen panel-numen text-ink flex flex-col p-1.5 font-sans text-base"
      role="menu"
      tabindex="-1"
      :aria-label="name"
      :style="menuStyle"
      @keydown="onKey"
    >
      <div v-if="searchable" class="menu__bar">
        <input
          ref="search"
          v-model="query"
          class="menu__search text-ink"
          type="search"
          autocomplete="off"
          spellcheck="false"
          :aria-label="searchName"
          :placeholder="searchName"
        />
      </div>

      <MenuRow
        v-for="(item, index) in rows"
        :ref="(row) => holdRow(item.id, row)"
        :key="item.id"
        :item="item"
        :current="current"
        :has-name="hasGroups && Boolean(item.group) && (item.isRule || index === 0)"
        :has-icons="Boolean($slots.icon)"
        @focus="here = index"
        @choose="choose(item)"
      >
        <slot name="icon" :id="item.id" />
      </MenuRow>

      <p v-if="!visibleItems.length" class="menu__silence text-hushed px-2 py-1.5">
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
  /* The room the area leaves it, set by the component. */
  --room: calc(100dvh - 1rem);
  /* A menu is as long as what it offers up to this, and its items scroll
     beyond it. */
  --tallest: min(24rem, var(--room));
  --lift: var(--numen-lift-menu, 99999);

  position: fixed;
  z-index: var(--lift, 99999);
  /* As wide as the longest thing it offers, within these two widths — or as
     wide as what asked for it, where that is wider than either. */
  inline-size: max-content;
  min-inline-size: max(var(--narrowest), var(--asking));
  max-inline-size: max(var(--widest), var(--asking));
  max-block-size: var(--tallest);
  overflow-y: auto;
  overscroll-behavior: contain;
  background-color: var(--numen-raised, #27221d);
  border: 1px solid var(--numen-panel-border, var(--numen-rule, rgba(255, 255, 255, 0.15)));
  border-radius: var(--numen-radius-panel, 0.75rem);
  box-shadow: var(--numen-panel-shadow, 0 12px 32px rgba(0, 0, 0, 0.5));
}

.menu:focus-visible {
  outline: none;
}

.menu__silence {
  margin: 0;
}

/* The search line is a row of its own at the top of the menu: it stays there
   while the items scroll beneath, and no item shows above or beside it. */
.menu__bar {
  /* The room the menu keeps around what it holds. */
  --inset: 0.375rem;

  position: sticky;
  inset-block-start: calc(-1 * var(--inset));
  z-index: 1;
  flex-shrink: 0;
  margin: calc(-1 * var(--inset)) calc(-1 * var(--inset)) 0.25rem;
  padding: var(--inset);
  background-color: var(--numen-raised, #27221d);
  border-block-end: 1px solid
    var(--numen-panel-border, var(--numen-rule, rgba(255, 255, 255, 0.15)));
}

.menu__search {
  inline-size: 100%;
  padding: 0.375rem 0.5rem;
  font: inherit;
  background-color: var(--numen-raised, #27221d);
  border: 1px solid var(--numen-field-rule, var(--numen-rule, rgba(255, 255, 255, 0.15)));
  border-radius: var(--numen-radius-small, 4px);
  outline: none;
}

.menu__search:focus-visible {
  border-color: var(--numen-accent, currentColor);
}
</style>
