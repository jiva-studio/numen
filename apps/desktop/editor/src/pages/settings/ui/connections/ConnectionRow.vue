<script setup lang="ts">
/**
 * One connection as a row of the list, drawn by the navigation row of the plugin manager: the
 * name, the type as the secondary text (with the address, or the path of a detected connection, and the
 * plugins that use it), and a chip for the state. The whole row opens the connection for editing. A connection detected on this computer
 * is marked Discovered and is no button: it has nothing to edit and cannot be deleted.
 */
import { computed, useTemplateRef } from 'vue'
import { NavigationRow } from '@numen/plugin-manager'
import type { ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import { getLocationText, getConnectionState, getUsedByNames } from '@/entities/connection'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  connection: ConnectionInfo
  type?: ConnectionTypeInfo | undefined
}>()

const emit = defineEmits<{
  (e: 'edit', opener: HTMLElement): void
}>()

/* --------------------------------- State ---------------------------------- */
const state = computed(() => getConnectionState(props.connection))
const typeTitle = computed(() => props.type?.title ?? props.connection.typeId)
const location = computed(() => getLocationText(props.type, props.connection))
const usedBy = computed(() => getUsedByNames(props.connection.uses))
const row = useTemplateRef<InstanceType<typeof NavigationRow>>('row')
const detail = computed(() =>
  [typeTitle.value, location.value, usedBy.value === '' ? '' : words.connectionUsedBy(usedBy.value)]
    .filter((one) => one !== '')
    .join(' · '),
)

/* -------------------------------- Handlers -------------------------------- */
function onEdit(): void {
  const element = row.value?.$el
  if (element instanceof HTMLElement) emit('edit', element)
}
</script>

<template>
  <NavigationRow
    ref="row"
    category="connections"
    :name="connection.name"
    :detail="detail"
    :is-static="connection.isDetected"
    :role="connection.isDetected ? 'listitem' : undefined"
    @select="onEdit"
  >
    <template #trail>
      <span v-if="connection.isDetected" class="connection-row__chip">
        {{ words.connectionDetected }}
      </span>
      <span v-if="state === 'unavailable'" class="connection-row__chip connection-row__chip--bad">
        {{ words.connectionUnavailable }}
      </span>
      <template v-else>
        <span v-if="state === 'incomplete'" class="connection-row__chip connection-row__chip--warn">
          {{ words.connectionIncomplete }}
        </span>
      </template>
    </template>
  </NavigationRow>
</template>

<style scoped>
.connection-row__chip {
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-tight);
  background-color: var(--numen-bubble-bg);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
}

.connection-row__chip--bad {
  background-color: var(--numen-alarm-bg);
  color: var(--numen-alarm);
}

.connection-row__chip--warn {
  color: var(--numen-ink);
}
</style>
