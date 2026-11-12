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
  (event: 'run-action', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const taskList = computed(() => orderTasks(props.tasks))
</script>

<template>
  <div class="task-popover" role="dialog" aria-modal="false" :aria-label="words.title">
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
        @run-action="(id) => emit('run-action', id)"
      />
    </div>
  </div>
</template>

<style scoped>
.task-popover {
  --popover-width: 24rem;
  --popover-edge-gap: 1rem;
  --popover-max-height: 22rem;
  --header-pad-block: 0.5rem;
  --header-pad-inline: 0.75rem;
  --close-size: 1.25rem;
  --body-pad: 0.5rem;
  --body-gap: 0.125rem;
  --empty-pad-block: 1.5rem;
  --empty-pad-inline: 1rem;
  display: flex;
  flex-direction: column;
  width: var(--popover-width);
  max-width: calc(100vw - var(--popover-edge-gap));
  max-height: var(--popover-max-height);
  background: var(--numen-raised);
  color: var(--numen-ink);
  border: 1px solid var(--numen-panel-border);
  border-radius: var(--numen-radius-popup);
  box-shadow: var(--numen-panel-shadow);
  overflow: hidden;
}

.task-popover__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--header-pad-block) var(--header-pad-inline);
  background: var(--numen-surface);
  border-bottom: 1px solid var(--numen-rule);
}

.task-popover__title {
  font-size: var(--numen-text-2);
  font-weight: 600;
  color: var(--numen-ink);
}

.task-popover__close-btn {
  width: var(--close-size);
  height: var(--close-size);
  padding: 0;
  color: var(--numen-hushed);
}

.task-popover__close-btn:hover {
  color: var(--numen-ink);
}

.task-popover__body {
  padding: var(--body-pad);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--body-gap);
}

.task-popover__empty {
  padding: var(--empty-pad-block) var(--empty-pad-inline);
  text-align: center;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}
</style>
