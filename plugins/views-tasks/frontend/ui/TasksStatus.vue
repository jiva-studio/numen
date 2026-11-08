<script setup lang="ts">
/**
 * The summary of the background tasks of the window, Ready or how many are active, with a badge of
 * the failed ones, and the list of the tasks it opens.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { BackgroundTask } from '@numen/plugin-sdk'
import { summarizeTasks } from '../lib/summary'
import { WORDS as words } from '../lib/words'
import CircularProgress from './CircularProgress.vue'
import TaskPopover from './TaskPopover.vue'
import './tasksStatus.css'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  tasks: readonly BackgroundTask[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
  (event: 'open-settings', pluginId: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose)

const summary = computed(() => summarizeTasks(props.tasks))
const summaryText = computed(() =>
  summary.value.activeCount === 0 ? words.idle : `${summary.value.activeCount} ${words.active}`,
)
const label = computed(() =>
  summary.value.failedCount > 0
    ? `${summaryText.value}, ${summary.value.failedCount} ${words.failed}`
    : summaryText.value,
)

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => (open ? dismissal.listen() : dismissal.release()))

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}
</script>

<template>
  <div
    ref="root"
    class="tasks-status"
    :class="{ 'tasks-status--active': summary.activeCount > 0 }"
    data-tasks-status
  >
    <button
      type="button"
      class="tasks-status__trigger"
      :class="{
        'tasks-status__trigger--busy': summary.activeCount > 0,
        'tasks-status__trigger--failed': summary.failedCount > 0,
      }"
      :aria-label="label"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="onToggle"
    >
      <div class="tasks-status__icon-wrap">
        <CircularProgress
          v-if="summary.activeCount > 0"
          :progress="summary.progress ?? 0"
          :size="14"
          :stroke-width="2"
        />
        <span v-else class="tasks-status__idle-dot" />
      </div>

      <span class="tasks-status__summary">{{ summaryText }}</span>

      <span v-if="summary.failedCount > 0" class="tasks-status__badge tasks-status__badge--alarm">
        {{ summary.failedCount }}
      </span>
    </button>

    <div v-if="isOpen" class="tasks-status__popover-container">
      <TaskPopover
        :tasks="tasks"
        @close="onClose"
        @cancel="(id) => emit('cancel', id)"
        @retry="(id) => emit('retry', id)"
        @dismiss="(id) => emit('dismiss', id)"
        @open-settings="(pluginId) => emit('open-settings', pluginId)"
      />
    </div>
  </div>
</template>
