<script setup lang="ts">
/**
 * Popover menu for linking a tab with another eligible open tab or unlinking it.
 */
import { computed, type Component } from 'vue'
import { Menu } from '@numen/ui'
import { ExternalLink, Link2Off } from '@lucide/vue'
import type { TabId } from '../types'
import { createLinkMenuItems, DETACH_ID, UNLINK_ID } from '../lib/menu'
import type { TabLinkMenuProps } from './tabLinkMenu.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<TabLinkMenuProps>(), {
  isLinked: false,
  linkedTargetTitle: '',
  canDetach: false,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'link', targetId: TabId): void
  (event: 'unlink'): void
  (event: 'detach'): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const items = computed(() =>
  createLinkMenuItems(props.eligibleTabs, props.isLinked, props.linkedTargetTitle, props.canDetach),
)

/* -------------------------------- Handlers -------------------------------- */
function onChoose(id: string) {
  if (id === DETACH_ID) {
    emit('detach')
  } else if (id === UNLINK_ID) {
    emit('unlink')
  } else {
    emit('link', id)
  }
}

function onDismiss() {
  emit('close')
}

/* -------------------------------- Helpers --------------------------------- */
function getIconFor(id: string): Component | null {
  if (id === DETACH_ID) return ExternalLink
  if (id === UNLINK_ID) return Link2Off
  const tab = props.eligibleTabs.find((one) => one.id === id)
  if (!tab) return null
  return props.getTabIcon(id) ?? props.getKindIcon?.(tab.kind) ?? null
}
</script>

<template>
  <Menu :items="items" :at="props.at" has-groups is-open @choose="onChoose" @dismiss="onDismiss">
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
