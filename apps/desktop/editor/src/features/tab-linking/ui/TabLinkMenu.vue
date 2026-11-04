<script setup lang="ts">
/**
 * Popover menu of a tab: detach it, move it to another window, link it with another eligible
 * open tab, or unlink it.
 */
import { computed, type Component } from 'vue'
import { Menu } from '@numen/ui'
import { AppWindow, ExternalLink, Link2Off } from '@lucide/vue'
import type { TabId } from '../types'
import { createLinkMenuItems, DETACH_ID, getMoveTarget, MOVE_ID, UNLINK_ID } from '../lib/menu'
import type { TabLinkMenuProps } from './tabLinkMenu.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<TabLinkMenuProps>(), {
  isLinked: false,
  linkedTargetTitle: '',
  canDetach: false,
  windows: () => [],
  isAsking: false,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'link', targetId: TabId): void
  (event: 'unlink'): void
  (event: 'detach'): void
  (event: 'move', windowId: string): void
  (event: 'highlight', windowId: string | null): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const items = computed(() => createLinkMenuItems(props))

/* -------------------------------- Handlers -------------------------------- */
function onChoose(id: string) {
  const windowId = getMoveTarget(id)
  if (windowId !== null) {
    emit('move', windowId)
  } else if (id === DETACH_ID) {
    emit('detach')
  } else if (id === UNLINK_ID) {
    emit('unlink')
  } else {
    emit('link', id)
  }
}

function onHighlight(id: string | null) {
  emit('highlight', getMoveTarget(id))
}

function onDismiss() {
  emit('close')
}

/* -------------------------------- Helpers --------------------------------- */
function getIconFor(id: string): Component | null {
  if (id === DETACH_ID) return ExternalLink
  if (id === UNLINK_ID) return Link2Off
  if (id === MOVE_ID) return AppWindow
  const windowId = getMoveTarget(id)
  if (windowId !== null) return getWindowIcon(windowId)
  const tab = props.eligibleTabs.find((one) => one.id === id)
  if (!tab) return null
  return props.getTabIcon(id) ?? props.getKindIcon?.(tab.kind) ?? null
}

/** Returns the kind icon of the active tab of a window. */
function getWindowIcon(windowId: string): Component | null {
  const kind = props.windows.find((one) => one.id === windowId)?.activeKind
  return kind ? (props.getKindIcon?.(kind) ?? null) : null
}
</script>

<template>
  <Menu
    :items="items"
    :at="props.at"
    has-groups
    is-open
    @choose="onChoose"
    @highlight="onHighlight"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <component
        :is="getIconFor(id)"
        v-if="getIconFor(id)"
        class="tab-link-menu__icon"
        aria-hidden="true"
      />
    </template>
    <template #silence>No open tabs to link with</template>
  </Menu>
</template>

<style scoped>
.tab-link-menu__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.75;
}
</style>
