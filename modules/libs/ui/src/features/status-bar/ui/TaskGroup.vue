<script setup lang="ts">
/**
 * Group of background tasks with category header.
 */
import type { TaskGroupData } from '../model/types'
import TaskItem from './TaskItem.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  group: TaskGroupData
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
}>()

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
</script>

<template>
  <div class="task-group">
    <div class="task-group__header">
      <span class="task-group__title">{{ group.name }}</span>
      <span class="task-group__count">{{ group.tasks.length }}</span>
    </div>

    <div class="task-group__items">
      <TaskItem
        v-for="task in group.tasks"
        :key="task.id"
        :task="task"
        @cancel="onCancel"
        @retry="onRetry"
        @dismiss="onDismiss"
      />
    </div>
  </div>
</template>

<style scoped>
.task-group {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.task-group__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.25rem 0.5rem;
  font-size: var(--numen-text-1);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--numen-hushed);
  border-bottom: 1px solid var(--numen-rule);
}

.task-group__count {
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}

.task-group__items {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}
</style>
