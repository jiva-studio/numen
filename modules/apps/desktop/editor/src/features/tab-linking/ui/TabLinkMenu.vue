<script setup lang="ts">
/**
 * Popover menu for linking a tab with another eligible open tab or unlinking it.
 */
import { computed } from 'vue'
import { Menu, type Position } from '@numen/ui'
import { Link2Off, type LucideIcon } from '@lucide/vue'
import { iconOfKind } from '@/entities/tab'
import type { EligibleLinkTab, TabId } from '../types'
import { createLinkMenuItems, UNLINK_ID } from '../lib/menu'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    tabId: TabId
    eligibleTabs: readonly EligibleLinkTab[]
    at: Position
    isLinked?: boolean
    linkedTargetTitle?: string
  }>(),
  { isLinked: false, linkedTargetTitle: '' },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'link', targetId: TabId): void
  (event: 'unlink'): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const items = computed(() =>
  createLinkMenuItems(props.eligibleTabs, props.isLinked, props.linkedTargetTitle),
)

/* -------------------------------- Handlers -------------------------------- */
function onChoose(id: string) {
  if (id === UNLINK_ID) {
    emit('unlink')
  } else {
    emit('link', id)
  }
}

function onDismiss() {
  emit('close')
}

/* -------------------------------- Helpers --------------------------------- */
function getIconFor(id: string): LucideIcon | null {
  if (id === UNLINK_ID) return Link2Off
  const tab = props.eligibleTabs.find((t) => t.id === id)
  return tab ? iconOfKind(tab.kind) : null
}
</script>

<template>
  <Menu :items="items" :at="props.at" has-groups open @choose="onChoose" @dismiss="onDismiss">
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
