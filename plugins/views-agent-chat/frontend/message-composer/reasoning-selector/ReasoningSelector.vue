<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed, ref, useTemplateRef } from 'vue'
import { Menu, type MenuItem, measureRect } from '@numen/ui'
import type { Position } from '@numen/ui'

export interface ReasoningLevelOption {
  readonly id: string
  readonly title: string
}

const props = withDefaults(
  defineProps<{
    levels?: readonly ReasoningLevelOption[]
    selectedId?: string
    /** What the control is called: its label while no level is chosen, and its menu's name. */
    name?: string
    isDisabled?: boolean
  }>(),
  {
    levels: () => [],
    selectedId: '',
    name: 'Reasoning',
    isDisabled: false,
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'select', id: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const trigger = useTemplateRef<HTMLButtonElement>('trigger')
const menuPosition = ref<Position>({ x: 0, y: 0 })

const label = computed(
  () => props.levels.find((one) => one.id === props.selectedId)?.title ?? props.name,
)

const menuItems = computed<MenuItem[]>(() =>
  props.levels.map((one) => ({ id: one.id, text: one.title, isDisabled: false })),
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  if (props.isDisabled || props.levels.length === 0) return
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  const rect = measureRect(trigger.value)
  if (rect) {
    menuPosition.value = { x: rect.left, y: rect.bottom + 4 }
  }
  isOpen.value = true
}

function onChoose(id: string) {
  isOpen.value = false
  if (props.levels.some((one) => one.id === id)) emit('select', id)
}

function onDismiss() {
  isOpen.value = false
}
</script>

<template>
  <div class="reasoning-selector relative inline-flex max-w-full min-w-0 shrink-0 items-center">
    <button
      ref="trigger"
      type="button"
      class="reasoning-selector__pill bg-bubble text-ink text-small hover:bg-surface rounded-pill inline-flex h-7 max-w-full min-w-0 cursor-pointer items-center gap-1.5 px-2.5 font-medium transition-colors disabled:pointer-events-none disabled:opacity-50"
      :aria-label="name"
      :disabled="isDisabled || levels.length === 0"
      @click="onToggle"
    >
      <span class="reasoning-selector__label min-w-0 flex-1 truncate">{{ label }}</span>
      <svg
        class="reasoning-selector__chevron h-3.5 w-3.5 shrink-0 opacity-60"
        viewBox="0 0 16 16"
        fill="currentColor"
      >
        <path
          fill-rule="evenodd"
          d="M4.22 6.22a.75.75 0 0 1 1.06 0L8 8.94l2.72-2.72a.75.75 0 1 1 1.06 1.06l-3.25 3.25a.75.75 0 0 1-1.06 0L4.22 7.28a.75.75 0 0 1 0-1.06Z"
          clip-rule="evenodd"
        />
      </svg>
    </button>

    <Menu
      :is-open="isOpen"
      :items="menuItems"
      :at="menuPosition"
      :current="selectedId"
      :from="trigger"
      :name="name"
      @choose="onChoose"
      @dismiss="onDismiss"
    />
  </div>
</template>
