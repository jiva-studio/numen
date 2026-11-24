<script setup lang="ts">
/**
 * One device of the identity: its name, when it was linked, its fingerprint and whether it is this
 * one. A right-click (or the Menu key) on another device opens a menu with Remove; this one has none.
 */
import { computed, ref } from 'vue'
import { Trash2 } from '@lucide/vue'
import { NavigationRow } from '@numen/plugin-manager'
import { Fingerprint, Menu, measureRect, type MenuItem } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ device: DeviceInfo }>()

const emit = defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()

/* --------------------------------- State ---------------------------------- */
const date = computed(() => {
  const added = new Date(props.device.addedAt)
  if (Number.isNaN(added.getTime())) return ''
  return new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(added)
})

const hasMenu = computed(() => !props.device.isThis && !props.device.isRevoked)
const row = ref<HTMLElement | null>(null)
const anchor = ref<{ x: number; y: number } | null>(null)

const items: readonly MenuItem[] = [{ id: 'revoke', text: words.deviceRevoke }]

/* -------------------------------- Handlers -------------------------------- */
function onContextMenu(event: MouseEvent): void {
  if (!hasMenu.value) return
  event.preventDefault()
  const box = measureRect(row.value)
  // The Menu key and Shift+F10 fire the event at 0,0.
  const isFromKeyboard = event.clientX === 0 && event.clientY === 0 && box !== undefined
  anchor.value = isFromKeyboard
    ? { x: box.left, y: box.bottom }
    : { x: event.clientX, y: event.clientY }
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
  <li
    ref="row"
    class="device-row"
    :tabindex="hasMenu ? 0 : undefined"
    :aria-haspopup="hasMenu ? 'menu' : undefined"
    @contextmenu="onContextMenu"
  >
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
        <Fingerprint class="device-row__fingerprint" :value="device.fingerprint" />
      </template>
    </NavigationRow>
    <Menu
      v-if="anchor"
      :items="items"
      :at="anchor"
      :from="row"
      :name="words.deviceMenu(device.name)"
      is-open
      @choose="onChoose"
      @dismiss="onClose"
    >
      <template #icon>
        <Trash2 :size="14" aria-hidden="true" />
      </template>
    </Menu>
  </li>
</template>

<style scoped>
.device-row__fingerprint {
  flex: 0 1 auto;
  min-inline-size: 8rem;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
}

.device-row__badge {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
}
</style>
