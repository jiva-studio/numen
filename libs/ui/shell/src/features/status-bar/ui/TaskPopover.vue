<script setup lang="ts">
/**
 * Popover displaying details of background tasks grouped by source category.
 */
import { computed } from 'vue'
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
const groups = computed(() => groupTasks(taskList.value))

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
      <button type="button" class="task-popover__close-btn" aria-label="Close" @click="onClose" />
    </div>

    <div v-if="groups.length === 0" class="task-popover__empty">
      {{ emptyLabel }}
    </div>

    <div v-else class="task-popover__body">
      <template v-for="group in groups" :key="group.name">
        <TaskGroup
          v-if="group.tasks.length > 1"
          :group="group"
          @cancel="onCancel"
          @retry="onRetry"
          @dismiss="onDismiss"
          @action="onAction"
        />
        <TaskItem
          v-for="task in group.tasks"
          v-else
          :key="task.id"
          :task="task"
          @cancel="onCancel"
          @retry="onRetry"
          @dismiss="onDismiss"
          @action="onAction"
        />
      </template>
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
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius-panel);
  box-shadow:
    0 10px 25px -5px var(--numen-backdrop),
    0 8px 10px -6px var(--numen-backdrop);
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
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.25rem;
  height: 1.25rem;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--numen-hushed);
  border-radius: var(--numen-radius-tight);
  cursor: pointer;
}

.task-popover__close-btn::before,
.task-popover__close-btn::after {
  content: '';
  position: absolute;
  width: 10px;
  height: 1.5px;
  background-color: currentColor;
}

.task-popover__close-btn::before {
  transform: rotate(45deg);
}

.task-popover__close-btn::after {
  transform: rotate(-45deg);
}

.task-popover__close-btn:hover {
  color: var(--numen-ink);
  background: var(--numen-rule);
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
