<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed, ref, useTemplateRef } from 'vue'
import { Menu, type MenuItem } from '@/shared/ui/menu'
import type { Position } from '@/shared/lib/geometry'

export interface AgentModelOption {
  readonly agentId: string
  readonly agentTitle: string
  readonly modelId: string
  readonly modelTitle: string
  readonly isAvailable: boolean
  readonly isDefault?: boolean
}

const props = withDefaults(
  defineProps<{
    options?: readonly AgentModelOption[]
    selectedAgentId?: string
    selectedModelId?: string
    disabled?: boolean
  }>(),
  {
    options: () => [],
    selectedAgentId: '',
    selectedModelId: '',
    disabled: false,
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'select', option: AgentModelOption): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const trigger = useTemplateRef<HTMLButtonElement>('trigger')
const menuPosition = ref<Position>({ x: 0, y: 0 })

const currentKey = computed(() => `${props.selectedAgentId}:${props.selectedModelId}`)

const currentOption = computed(() => {
  const match = props.options.find(
    (one) => one.agentId === props.selectedAgentId && one.modelId === props.selectedModelId,
  )
  if (match) return match
  return props.options.find((one) => one.isDefault) ?? props.options[0]
})

const label = computed(() => {
  const active = currentOption.value
  if (!active) return 'No agent installed'
  return `${active.agentTitle} · ${active.modelTitle}`
})

const menuItems = computed<MenuItem[]>(() =>
  props.options.map((one) => ({
    id: `${one.agentId}:${one.modelId}`,
    text: one.modelTitle,
    group: one.agentTitle,
    disabled: false,
  })),
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  if (props.disabled || props.options.length === 0) return
  if (isOpen.value) {
    isOpen.value = false
    return
  }
  const rect = trigger.value?.getBoundingClientRect()
  if (rect) {
    menuPosition.value = { x: rect.left, y: rect.bottom + 4 }
  }
  isOpen.value = true
}

function onChoose(id: string) {
  isOpen.value = false
  const chosen = props.options.find((one) => `${one.agentId}:${one.modelId}` === id)
  if (chosen) emit('select', chosen)
}

function onDismiss() {
  isOpen.value = false
}
</script>

<template>
  <div class="model-selector relative inline-flex max-w-full min-w-0 items-center">
    <button
      ref="trigger"
      type="button"
      class="model-selector__pill bg-bubble text-ink text-small hover:bg-surface rounded-pill inline-flex h-7 max-w-full min-w-0 cursor-pointer items-center gap-1.5 px-2.5 font-medium transition-colors disabled:pointer-events-none disabled:opacity-50"
      :disabled="disabled || options.length === 0"
      @click="onToggle"
    >
      <span class="model-selector__label min-w-0 flex-1 truncate">{{ label }}</span>
      <svg
        class="model-selector__chevron h-3.5 w-3.5 shrink-0 opacity-60"
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
      :open="isOpen"
      :items="menuItems"
      :at="menuPosition"
      :current="currentKey"
      :from="trigger"
      :has-groups="true"
      name="Model"
      @choose="onChoose"
      @dismiss="onDismiss"
    />
  </div>
</template>
