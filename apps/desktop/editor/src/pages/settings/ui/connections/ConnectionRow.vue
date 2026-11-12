<script setup lang="ts">
/**
 * One connection as a row of the list, drawn by the row of the plugin list: the name, the type as
 * the secondary text (with the address and the plugins that use it), and a chip for each operation or
 * for the state. The whole row opens the connection for editing.
 */
import { computed, useTemplateRef } from 'vue'
import { SystemRow } from '@numen/plugin-manager'
import type { ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import {
  getAddressText,
  getConnectionState,
  getOperationTitles,
  getUsedByNames,
} from '@/entities/connection'
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
const address = computed(() => getAddressText(props.type, props.connection))
const operations = computed(() => getOperationTitles(props.type))
const usedBy = computed(() => getUsedByNames(props.connection.uses))
const row = useTemplateRef<InstanceType<typeof SystemRow>>('row')
const detail = computed(() =>
  [
    typeTitle.value,
    address.value,
    usedBy.value === '' ? '' : words.connectionUsedBy(usedBy.value),
  ]
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
  <SystemRow ref="row" :name="connection.name" :detail="detail" @select="onEdit">
    <template #trail>
      <span v-if="state === 'unavailable'" class="connection-row__chip connection-row__chip--bad">
        {{ words.connectionUnavailable }}
      </span>
      <template v-else>
        <span v-if="state === 'incomplete'" class="connection-row__chip connection-row__chip--warn">
          {{ words.connectionIncomplete }}
        </span>
        <span
          v-for="operation in operations"
          :key="operation"
          class="connection-row__chip"
        >
          {{ operation }}
        </span>
      </template>
    </template>
  </SystemRow>
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
