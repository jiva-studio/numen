<script setup lang="ts">
/**
 * One device of the account: its name, when it was linked, its fingerprint and whether it is this
 * one. Every row has the same columns; the actions column holds a "..." menu with Remove on another
 * device and stays empty on this one.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Button, Menu, type MenuItem } from '@numen/ui'
import { NavigationRow } from '@numen/plugin-manager'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ device: DeviceInfo }>()

const emit = defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof Button>>('trigger')
const anchor = ref<{ x: number; y: number; wide: number } | null>(null)

const date = computed(() => {
  const added = new Date(props.device.addedAt)
  if (Number.isNaN(added.getTime())) return ''
  return new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(added)
})

const hasActions = computed(() => !props.device.isThis && !props.device.isRevoked)
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
  const box = getTrigger()?.getBoundingClientRect()
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
  <li class="device-row">
    <NavigationRow
      category="device"
      is-static
      :name="device.name"
      :detail="words.deviceLinked(date)"
    >
      <template #name-suffix>
        <span v-if="device.isThis" class="device-row__badge">{{ words.deviceThis }}</span>
      </template>
      <template #trail>
        <span class="device-row__fingerprint">{{ device.fingerprint }}</span>
        <span class="device-row__actions">
          <Button
            v-if="hasActions"
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
        </span>
      </template>
    </NavigationRow>
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
  </li>
</template>

<style scoped>
.device-row__fingerprint {
  flex: none;
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.device-row__actions {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
}

.device-row__badge {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
}
</style>
