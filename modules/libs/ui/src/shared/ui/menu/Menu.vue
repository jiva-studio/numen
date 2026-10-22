<script setup lang="ts">
/**
 * A menu: a list of things that can be chosen, put where it was asked for.
 *
 * it, and it is placed against the area it is drawn into. It takes items and a
 * point and says which item was chosen; what the items are and what choosing
 * one does are the caller's.
 */
import { computed, nextTick, useTemplateRef, watch } from 'vue'
import MenuRow from './MenuRow.vue'
import { getInitialIndex, groupItems, type MenuItem } from './items/item'
import { useMenuDismissal } from './interaction/menuDismissal'
import { useMenuKeys } from './interaction/keys'
import { useMenuOpen } from './interaction/open'
import { useMenuPlacement } from './interaction/place'
import { useMenuSearch } from './items/search'
import type { MenuEvents, MenuProps, MenuSlots } from './Menu.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<MenuProps>(), {
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
  isSearchable: false,
  searchName: 'Search',
  clock: () => Date.now(),
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<MenuEvents>()

defineSlots<MenuSlots>()

/* ---------------------------------- State --------------------------------- */
const menu = useTemplateRef<HTMLElement>('menu')
const search = useTemplateRef<HTMLInputElement>('search')

const { placed, setSize } = useMenuPlacement({
  at: () => props.at,
  viewport: () => props.viewport,
  margin: () => props.margin,
})

const { listen, release } = useMenuDismissal(menu, () => emit('dismiss'))

const { query, visibleItems, onKey, focusInitial } = useMenuSearch({
  isSearchable: () => props.isSearchable,
  items: () => props.items,
  searchRef: search,
  menuRef: menu,
  onRowKey: (event) => onRowKey(event),
  onChoose,
  onResetHere: () => {
    here.value = -1
  },
  goToInitial: () => goTo(getInitialIndex(props.trigger, props.items, props.current)),
})

/** The items shown, with the rules that stand between their groups. */
const rows = computed(() => groupItems(visibleItems.value))

const {
  here,
  holdRow,
  goTo,
  onKey: onRowKey,
} = useMenuKeys(() => visibleItems.value, menu, props.clock)

const isMenuVisible = computed(() => Boolean(props.isOpen))

const menuStyle = computed(() => ({
  left: `${placed.value.x}px`,
  top: `${placed.value.y}px`,
  '--asking': `${props.asking}px`,
  '--room': props.viewport
    ? `${Math.max(props.viewport.height - 2 * props.margin, 0)}px`
    : `calc(100dvh - ${2 * props.margin}px)`,
}))

/* ---------------------------------- Hooks --------------------------------- */
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

/** Measured again when what it holds changes, and when the point does. */
watch(
  () => [visibleItems.value, props.at],
  async () => {
    if (!isMenuVisible.value) return
    await nextTick()
    measure()
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onChoose(item: MenuItem): void {
  if (item.isDisabled) return
  emit('choose', item.id)
  emit('dismiss')
}

/* --------------------------------- Helpers -------------------------------- */
/** Its own size, which only the drawing knows. */
function measure(): void {
  const box = menu.value?.getBoundingClientRect()
  if (box) setSize({ width: box.width, height: box.height })
}
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
      <div v-if="isSearchable" class="menu__bar">
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
        @choose="onChoose(item)"
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
