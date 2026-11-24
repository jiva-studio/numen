<script setup lang="ts">
/** The "..." button of a device row and the menu it opens with Remove. */
import { ref, useTemplateRef } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Button, Menu, measureRect, type MenuItem } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ device: DeviceInfo }>()

const emit = defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof Button>>('trigger')
const anchor = ref<{ x: number; y: number; wide: number } | null>(null)

const items: readonly MenuItem[] = [{ id: 'revoke', text: words.deviceRevoke }]

const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

/* -------------------------------- Handlers -------------------------------- */
function onToggle(): void {
  if (anchor.value !== null) {
    onClose()
    return
  }
  const box = measureRect(getTrigger())
  if (box !== undefined) anchor.value = { x: box.left, y: box.bottom, wide: box.width }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(): void {
  onClose()
  emit('revoke', props.device)
}
</script>

<template>
  <Button
    ref="trigger"
    type="button"
    variant="ghost"
    size="icon-small"
    :aria-label="words.deviceMore(device.name)"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    @click="onToggle"
  >
    <Ellipsis :size="16" aria-hidden="true" />
  </Button>
  <Menu
    v-if="anchor"
    :items="items"
    :at="{ x: anchor.x, y: anchor.y }"
    :asking="anchor.wide"
    :from="getTrigger()"
    :name="words.deviceMore(device.name)"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="onClose"
  />
</template>
