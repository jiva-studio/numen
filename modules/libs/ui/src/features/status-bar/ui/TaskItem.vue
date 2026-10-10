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
}>()

/* --------------------------------- State ---------------------------------- */
const subtext = computed(() => {
  if (props.task.state === 'failed' && props.task.errorReason) {
    return props.task.errorReason
  }
  return props.task.detail || ''
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
</script>

<template>
  <div class="task-item" :class="`task-item--${task.state}`" :data-task-id="task.id">
    <div class="task-item__content">
      <span class="task-item__label">{{ task.label }}</span>
      <span
        v-if="subtext"
        class="task-item__detail"
        :class="{ 'text-alarm': task.state === 'failed' }"
      >
        {{ subtext }}
      </span>
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
