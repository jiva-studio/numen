<script setup lang="ts">
/**
 * Popover displaying details of background tasks grouped by source category.
 */
import { computed } from 'vue'
import { X } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { StatusBarTask } from '../model/types'
import { groupTasks } from '../lib/aggregate'
import TaskGroup from './TaskGroup.vue'
import TaskItem from './TaskItem.vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    tasks?: readonly StatusBarTask[] | undefined
    title?: string | undefined
    emptyLabel?: string | undefined
  }>(),
  {
    tasks: () => [],
    title: 'Background Tasks',
    emptyLabel: 'No active background tasks',
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'action', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const taskList = computed(() => props.tasks ?? [])
const listing = computed(() => groupTasks(taskList.value))
const isEmpty = computed(
  () => listing.value.coreTasks.length === 0 && listing.value.groups.length === 0,
)

/* -------------------------------- Handlers -------------------------------- */
function onCancel(id: string) {
  emit('cancel', id)
}

function onRetry(id: string) {
  emit('retry', id)
}

function onDismiss(id: string) {
  emit('dismiss', id)
}

function onAction(id: string) {
  emit('action', id)
}

function onClose() {
  emit('close')
}
</script>

<template>
  <div class="task-popover" role="dialog" aria-modal="false" :aria-label="title">
    <div class="task-popover__header">
      <span class="task-popover__title">{{ title }}</span>
      <Button
        variant="ghost"
        size="icon-small"
        class="task-popover__close-btn"
        title="Close"
        aria-label="Close"
        @click="onClose"
      >
        <X :size="12" aria-hidden="true" />
      </Button>
    </div>

    <div v-if="isEmpty" class="task-popover__empty">
      {{ emptyLabel }}
    </div>

    <div v-else class="task-popover__body">
      <TaskItem
        v-for="task in listing.coreTasks"
        :key="task.id"
        :task="task"
        @cancel="onCancel"
        @retry="onRetry"
        @dismiss="onDismiss"
        @action="onAction"
      />
      <TaskGroup
        v-for="group in listing.groups"
        :key="group.name"
        :group="group"
        @cancel="onCancel"
        @retry="onRetry"
        @dismiss="onDismiss"
        @action="onAction"
      />
    </div>
  </div>
</template>

<style scoped>
.task-popover {
  display: flex;
  flex-direction: column;
  width: 24rem;
  max-height: 22rem;
  background: var(--numen-raised);
  color: var(--numen-ink);
  border: 1px solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
  overflow: hidden;
  z-index: 50;
}

.task-popover__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.75rem;
  background: var(--numen-surface);
  border-bottom: 1px solid var(--numen-rule);
}

.task-popover__title {
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.task-popover__close-btn {
  width: 1.25rem;
  height: 1.25rem;
  padding: 0;
  color: var(--numen-hushed);
}

.task-popover__close-btn:hover {
  color: var(--numen-ink);
}

.task-popover__body {
  padding: 0.5rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.task-popover__empty {
  padding: 1.5rem 1rem;
  text-align: center;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}
</style>
