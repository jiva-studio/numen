<script setup lang="ts">
/**
 * A stack of tabs and the one of them that is showing.
 *
 * The strip and the group carry their identities on the element, so that a
 * drag can find out what the pointer is over by asking the document.
 */
import WorkspaceTab from './WorkspaceTab.vue'
import type { Group, TabId } from '../model'

defineProps<{
  group: Group
  /** What each tab is called. A tab with no title is shown by its identity. */
  titles: Readonly<Record<TabId, string>>
  /** The group a tab would open into. */
  focused?: boolean
}>()

const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, at: PointerEvent): void
  (event: 'take'): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  silence(): unknown
}>()
</script>

<template>
  <section
    class="group numen flex min-h-0 min-w-0 flex-col bg-surface text-ink"
    :data-workspace-group="group.id"
    :data-focused="focused || undefined"
    @pointerdown="emit('take')"
  >
    <div
      class="group__strip flex shrink-0 items-stretch overflow-hidden"
      role="tablist"
      :data-workspace-strip="group.id"
    >
      <WorkspaceTab
        v-for="tab in group.tabs"
        :key="tab"
        :data-workspace-tab="tab"
        :title="titles[tab] ?? tab"
        :showing="tab === group.active"
        @lift="emit('lift', tab, $event)"
        @close="emit('close', tab)"
        @click="emit('choose', tab)"
      />
    </div>

    <div class="group__body min-h-0 min-w-0 flex-1">
      <slot v-if="group.active" name="tab" :id="group.active" />
      <p v-else class="group__silence font-sans text-small text-hushed">
        <slot name="silence">Nothing open</slot>
      </p>
    </div>
  </section>
</template>

<style scoped>
.group__strip {
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
  background: color-mix(in oklab, var(--numen-surface) 88%, var(--numen-node-fg));
}

.group__body {
  position: relative;
  overflow: hidden;
}

.group__silence {
  display: grid;
  place-items: center;
  block-size: 100%;
  margin: 0;
  opacity: 0.6;
}
</style>
