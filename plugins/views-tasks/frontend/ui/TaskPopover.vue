<script setup lang="ts">
/**
 * The list of the background tasks, newest first.
 */
import { computed } from 'vue'
import { X } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { BackgroundTask } from '@numen/plugin-sdk'
import { orderTasks } from '../lib/summary'
import { WORDS as words } from '../lib/words'
import TaskItem from './TaskItem.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  tasks: readonly BackgroundTask[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'open-settings', pluginId: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const taskList = computed(() => orderTasks(props.tasks))
</script>

<template>
  <div class="task-popover">
    <div class="task-popover__header">
      <span class="task-popover__title">{{ words.title }}</span>
      <Button
        variant="ghost"
        size="icon-small"
        class="task-popover__close-btn"
        :title="words.close"
        :aria-label="words.close"
        @click="emit('close')"
      >
        <X :size="12" aria-hidden="true" />
      </Button>
    </div>

    <div v-if="taskList.length === 0" class="task-popover__empty">
      {{ words.empty }}
    </div>

    <div v-else class="task-popover__body">
      <TaskItem
        v-for="task in taskList"
        :key="task.id"
        :task="task"
        @cancel="(id) => emit('cancel', id)"
        @retry="(id) => emit('retry', id)"
        @dismiss="(id) => emit('dismiss', id)"
        @open-settings="(pluginId) => emit('open-settings', pluginId)"
      />
    </div>
  </div>
</template>

<style scoped>
.task-popover {
  display: flex;
  flex-direction: column;
  width: 24rem;
  max-width: calc(100vw - 1rem);
  max-height: 22rem;
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
  gap: 0.125rem;
}

.task-popover__empty {
  padding: 1.5rem 1rem;
  text-align: center;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}
</style>
