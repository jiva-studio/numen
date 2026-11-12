<script setup lang="ts">
/**
 * One connection as a row of a list, drawn like the rows of the plugin list: its name and type, the
 * address when the person entered one, what the type can do (a chip for each operation) or why the
 * connection cannot be used. The whole row is a button that opens the connection for editing.
 */
import { computed } from 'vue'
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

/* -------------------------------- Handlers -------------------------------- */
function onEdit(event: Event): void {
  if (event.currentTarget instanceof HTMLElement) emit('edit', event.currentTarget)
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  onEdit(event)
}
</script>

<template>
  <li
    class="connection-row"
    role="button"
    tabindex="0"
    :aria-label="connection.name"
    @click="onEdit"
    @keydown="onKeydown"
  >
    <div class="connection-row__main">
      <div class="connection-row__title">
        <span class="connection-row__name">{{ connection.name }}</span>
        <span class="connection-row__type">{{ typeTitle }}</span>
      </div>
      <div v-if="address !== ''" class="connection-row__address">{{ address }}</div>
      <div v-if="usedBy !== ''" class="connection-row__address">
        {{ words.connectionUsedBy(usedBy) }}
      </div>
    </div>
    <div class="connection-row__state">
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
          class="connection-row__chip connection-row__chip--operation"
        >
          {{ operation }}
        </span>
      </template>
    </div>
  </li>
</template>

<style scoped>
.connection-row {
  --row-min-size: 3.5rem;
  --row-pad-block: 0.625rem;
  --row-pad-inline: 0.75rem;
  --focus-width: 0.125rem;
  --focus-inset: -0.125rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  min-block-size: var(--row-min-size);
  padding: var(--row-pad-block) var(--row-pad-inline);
  border-radius: var(--numen-radius-tight);
  cursor: pointer;
  list-style: none;
  user-select: none;
  transition: background-color 0.15s ease;
}

.connection-row:hover {
  background-color: var(--numen-bubble-bg);
}

.connection-row:focus-visible {
  outline: var(--focus-width) solid var(--numen-accent);
  outline-offset: var(--focus-inset);
}

.connection-row__main {
  flex: 1 1 auto;
  min-inline-size: 0;
}

.connection-row__title {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: baseline;
}

.connection-row__name {
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.connection-row__type,
.connection-row__address {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.connection-row__address {
  overflow-wrap: anywhere;
  font-family: var(--numen-font-mono);
}

.connection-row__state {
  display: flex;
  flex-wrap: wrap;
  flex-shrink: 0;
  gap: 0.25rem;
  align-items: center;
  justify-content: flex-end;
}

.connection-row__chip {
  padding: 0.125rem 0.4375rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-pill);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  line-height: 1;
  white-space: nowrap;
}

.connection-row__chip--bad {
  border-color: var(--numen-alarm);
  color: var(--numen-alarm);
}

.connection-row__chip--warn {
  color: var(--numen-ink);
}
</style>
