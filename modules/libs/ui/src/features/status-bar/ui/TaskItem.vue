<script setup lang="ts">
/**
 * Single task row in the background tasks popover.
 */
import { computed } from 'vue'
import type { StatusBarTask } from '../model/types'
import CircularProgress from './CircularProgress.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  task: StatusBarTask
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'action', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const subtext = computed(() => {
  if (props.task.state === 'failed' && props.task.error) {
    return props.task.error
  }
  return props.task.about || ''
})

/* -------------------------------- Handlers -------------------------------- */
function onCancel() {
  emit('cancel', props.task.id)
}

function onRetry() {
  emit('retry', props.task.id)
}

function onDismiss() {
  emit('dismiss', props.task.id)
}

function onAction() {
  emit('action', props.task.id)
}
</script>

<template>
  <div class="task-item" :class="`task-item--${task.state}`" :data-task-id="task.id">
    <div class="task-item__content">
      <span class="task-item__label" :title="task.label">{{ task.label }}</span>
      <span
        v-if="subtext"
        class="task-item__detail"
        :class="{ 'text-alarm task-item__detail--wrapped': task.state === 'failed' }"
        :title="subtext"
      >
        {{ subtext }}
      </span>
      <div v-if="task.state === 'failed' && task.actionLabel" class="task-item__failure-row">
        <button type="button" class="task-item__action-link" @click.stop="onAction">
          {{ task.actionLabel }} &rarr;
        </button>
      </div>
    </div>

    <div class="task-item__action">
      <button
        v-if="task.state === 'running' && task.canCancel !== false"
        type="button"
        class="task-item__btn task-item__btn--cancel"
        title="Cancel task"
        aria-label="Cancel task"
        @click.stop="onCancel"
      />

      <button
        v-else-if="task.state === 'failed'"
        type="button"
        class="task-item__btn task-item__btn--retry"
        title="Retry task"
        aria-label="Retry task"
        @click.stop="onRetry"
      />

      <button
        v-else-if="task.state === 'completed'"
        type="button"
        class="task-item__btn task-item__btn--dismiss"
        title="Dismiss"
        aria-label="Dismiss"
        @click.stop="onDismiss"
      />

      <CircularProgress
        class="task-item__indicator"
        :progress="task.progress"
        :state="task.state"
        :size="18"
        :stroke-width="2.2"
      />
    </div>
  </div>
</template>
