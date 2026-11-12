<script setup lang="ts">
/**
 * A menu: a list of selectable items, positioned at the point it was requested.
 */
import { computed, useTemplateRef } from 'vue'
import MenuRow from './MenuRow.vue'
import { getNow, measureElementRect, measureRect } from '@/shared/api/dom'
import type { Position } from '@/shared/lib/math/geometry'
import { useMenuDismissal } from './interaction/menuDismissal'
import { useMenuLifecycle } from './interaction/useMenuLifecycle'
import { useMenuNavigation } from './interaction/useMenuNavigation'
import { useMenuPlacement } from './interaction/place'
import type { MenuEvents, MenuProps, MenuSlots } from './Menu.types'
import type { MenuItem } from './items/item'

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
  clock: () => getNow(),
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<MenuEvents>()

defineSlots<MenuSlots>()

defineExpose({ getElement: () => menu.value })

/* --------------------------------- State ---------------------------------- */
const menu = useTemplateRef<HTMLElement>('menu')
const submenuRef = useTemplateRef<{ getElement?: () => HTMLElement | null }>('submenuRef')

const { setSize, resetSize, menuStyle } = useMenuPlacement({
  at: () => props.at,
  viewport: () => props.viewport,
  margin: () => props.margin,
  asking: () => props.asking,
})

const isIgnored = (event: Event): boolean =>
  event.target instanceof Element &&
  Boolean(event.target.closest('.menu') && event.target.closest('.menu') !== menu.value)

const { listen, release } = useMenuDismissal(menu, () => emit('dismiss'), isIgnored)

const { visibleItems, rows, here, submenu, focusInitial, pointTo, onKey, onResetNavigation } =
  useMenuNavigation({
    items: () => props.items,
    trigger: () => props.trigger,
    current: () => props.current,
    clock: props.clock,
    menuRef: menu,
    getFallbackPosition: () => props.at,
    getSubmenuPosition: (el) => getSubmenuPosition(el),
    getSubmenuElement: () => submenuRef.value?.getElement?.() ?? null,
    getSubmenuBox: () => measureRect(submenuRef.value?.getElement?.()) ?? null,
    onEmitChoose: (id) => emit('choose', id),
    onEmitDismiss: () => emit('dismiss'),
    hasFrom: () => Boolean(props.from),
  })

const isMenuVisible = computed(() => Boolean(props.isOpen))

let highlightedId: string | null = null
let lastPointer: { x: number; y: number } | null = null

/* --------------------------------- Hooks ---------------------------------- */
useMenuLifecycle({
  isOpen: () => isMenuVisible.value,
  listen,
  release,
  measure,
  focusInitial,
  onReset: () => {
    resetSize()
    onResetNavigation()
  },
  getFrom: () => props.from,
  getWatchedDeps: () => [visibleItems.value, props.at],
})

/* -------------------------------- Handlers -------------------------------- */
function onRowHover(item: MenuItem, index: number, event: MouseEvent): void {
  submenu.onRowHover(item, event)
  onHighlight(item.id)
  // A row under a pointer that did not move (the list changed beneath it) is not taken.
  if (event.type !== 'mousemove') return
  const moved = !lastPointer || lastPointer.x !== event.clientX || lastPointer.y !== event.clientY
  lastPointer = { x: event.clientX, y: event.clientY }
  if (moved && !item.isDisabled) pointTo(index)
}

function onMenuLeave(event: MouseEvent): void {
  submenu.onMenuLeave(event)
  onHighlight(null)
  // An open submenu keeps its parent row active.
  if (!submenu.active.value) {
    here.value = -1
    menu.value?.focus({ preventScroll: true })
  }
}

function onHighlight(id: string | null): void {
  if (id === highlightedId) return
  highlightedId = id
  emit('highlight', id)
}

/* -------------------------------- Helpers --------------------------------- */
function getSubmenuPosition(el: HTMLElement | null): Position | null {
  const box = el ? measureElementRect(el) : undefined
  return box ? { x: box.right, y: box.top } : null
}

function measure(): void {
  const box = menu.value ? measureElementRect(menu.value) : undefined
  if (box) setSize({ width: box.width, height: box.height })
}
</script>

<template>
  <Teleport :to="to">
    <div
      v-if="isMenuVisible"
      ref="menu"
      class="menu numen"
      role="menu"
      tabindex="-1"
      :aria-label="name"
      :style="menuStyle"
      @keydown="onKey"
      @mouseleave="onMenuLeave"
    >
      <MenuRow
        v-for="(item, index) in rows"
        :ref="(row) => submenu.onRegisterRow(item.id, row)"
        :key="item.id"
        :item="item"
        :current="current"
        :has-name="hasGroups && Boolean(item.group) && (item.isRule || index === 0)"
        :has-icons="Boolean($slots.icon)"
        :is-active="index === here"
        @focus="here = index"
        @hover="(event) => onRowHover(item, index, event)"
        @choose="submenu.onChoose(item)"
      >
        <slot name="icon" :id="item.id" />
      </MenuRow>

      <p v-if="!visibleItems.length" class="menu__silence">
        <slot name="silence">No items</slot>
      </p>
    </div>

    <Menu
      v-if="submenu.active.value"
      ref="submenuRef"
      :items="submenu.active.value.items"
      :at="submenu.active.value.at"
      :from="submenu.active.value.from"
      :viewport="viewport"
      :margin="margin"
      :to="to"
      :clock="clock"
      :has-groups="hasGroups"
      :trigger="submenu.trigger.value"
      is-open
      @choose="submenu.onSubmenuChoose"
      @dismiss="submenu.onClose"
      @highlight="onHighlight"
    >
      <template v-if="$slots.icon" #icon="{ id }">
        <slot name="icon" :id="id" />
      </template>
    </Menu>
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
  --widest: 22rem;
  /* The width of the trigger, which is the menu's minimum width. */
  --asking: 0px;
  /* The space the available area leaves for the menu, set by the component. */
  --room: calc(100dvh - 1rem);
  /* A menu is as long as what it offers up to this, and its items scroll
     beyond it. */
  --tallest: min(24rem, var(--room));
  --lift: var(--numen-lift-menu);
  --inset: 0.375rem;
  --silence-pad: 0.375rem 0.5rem;

  box-sizing: border-box;
  position: fixed;
  z-index: var(--lift);
  display: flex;
  flex-direction: column;
  padding: var(--inset);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  color: var(--numen-ink);
  inline-size: max-content;
  min-inline-size: max(var(--narrowest), var(--asking), var(--opened-width));
  max-inline-size: max(var(--widest), var(--asking));
  max-block-size: var(--tallest);
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  background-color: var(--numen-raised);
  border: 1px solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
}

.menu:focus-visible {
  outline: none;
}

.menu__silence {
  margin: 0;
  padding: var(--silence-pad);
  color: var(--numen-hushed);
}
</style>
