<script setup lang="ts">
/**
 * Persistent bottom status bar displaying background task activity and details popover.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { StatusBarTask } from '../model/types'
import { calculateAggregate } from '../lib/aggregate'
import CircularProgress from './CircularProgress.vue'
import TaskPopover from './TaskPopover.vue'
import './statusBar.css'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    tasks?: readonly StatusBarTask[] | undefined
    idleLabel?: string | undefined
    workingLabel?: string | undefined
  }>(),
  { tasks: () => [], idleLabel: 'Ready', workingLabel: 'Background tasks' },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'cancel', id: string): void
  (event: 'retry', id: string): void
  (event: 'dismiss', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const rootRef = ref<HTMLElement | null>(null)

const taskList = computed(() => props.tasks ?? [])
const aggregate = computed(() => calculateAggregate(taskList.value))
const summaryText = computed(() => {
  if (aggregate.value.isIdle) return props.idleLabel
  return `${aggregate.value.activeCount} active`
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  document.addEventListener('click', onClickOutside)
  document.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  document.removeEventListener('keydown', onKeyDown)
})

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}
function onClose() {
  isOpen.value = false
}
function onCancel(id: string) {
  emit('cancel', id)
}
function onRetry(id: string) {
  emit('retry', id)
}
function onDismiss(id: string) {
  emit('dismiss', id)
}

function onClickOutside(event: MouseEvent) {
  if (isOpen.value && rootRef.value && !rootRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isOpen.value) isOpen.value = false
}
</script>

<template>
  <footer
    ref="rootRef"
    class="status-bar"
    :class="{ 'status-bar--active': !aggregate.isIdle }"
    role="region"
    aria-label="Status Bar"
  >
    <button
      type="button"
      class="status-bar__trigger"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="onToggle"
    >
      <div class="status-bar__icon-wrap">
        <CircularProgress
          v-if="!aggregate.isIdle"
          :progress="aggregate.overallProgress"
          :size="14"
          :stroke-width="2"
        />
        <span v-else class="status-bar__idle-dot" />
      </div>

      <span class="status-bar__summary">{{ summaryText }}</span>

      <span v-if="aggregate.failedCount > 0" class="status-bar__badge status-bar__badge--alarm">
        {{ aggregate.failedCount }} failed
      </span>
    </button>

    <div v-if="isOpen" class="status-bar__popover-container">
      <TaskPopover
        :tasks="taskList"
        :title="workingLabel"
        @close="onClose"
        @cancel="onCancel"
        @retry="onRetry"
        @dismiss="onDismiss"
      />
    </div>
  </footer>
</template>
