<script setup lang="ts">
/**
 * One connection as a row: its name and type, the address when the person entered one, what the type
 * can do or why the connection cannot be used, and the menu to edit and delete it.
 */
import { computed, ref } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Button, Menu, measureRect, measureWindow, type MenuItem } from '@numen/ui'
import type { ConnectionInfo, ConnectionTypeInfo } from '@/entities/connection'
import {
  getAddressText,
  getConnectionState,
  getOperationsText,
  getUsedByNames,
} from '@/entities/connection'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  connection: ConnectionInfo
  type?: ConnectionTypeInfo | undefined
}>()

const emit = defineEmits<{
  (e: 'edit'): void
  (e: 'delete', opener: HTMLElement): void
}>()

/* --------------------------------- State ---------------------------------- */
const MARGIN = 8
const items: readonly MenuItem[] = [
  { id: 'edit', text: words.connectionMenuEdit },
  { id: 'delete', text: words.connectionMenuDelete },
]
const state = computed(() => getConnectionState(props.connection))
const typeTitle = computed(() => props.type?.title ?? props.connection.typeId)
const address = computed(() => getAddressText(props.type, props.connection))
const operations = computed(() => getOperationsText(props.type))
const usedBy = computed(() => getUsedByNames(props.connection.uses))
const anchor = ref<{ x: number; y: number; height: number } | null>(null)
const opener = ref<HTMLElement | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpen(event: MouseEvent): void {
  if (!(event.currentTarget instanceof HTMLElement)) return
  const box = measureRect(event.currentTarget)
  opener.value = event.currentTarget
  anchor.value = { x: box.right, y: box.bottom, height: measureWindow().height }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  const from = opener.value
  onClose()
  if (id === 'edit') emit('edit')
  else if (id === 'delete' && from) emit('delete', from)
}
</script>

<template>
  <li class="connection-row" :aria-label="connection.name">
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
        <span v-if="operations !== ''" class="connection-row__chip">{{ operations }}</span>
      </template>
      <Button
        variant="outline"
        size="icon"
        class="connection-row__menu"
        aria-haspopup="menu"
        :aria-expanded="anchor !== null"
        :aria-label="words.connectionActions"
        :title="words.connectionActions"
        @click="onOpen"
      >
        <Ellipsis :size="14" aria-hidden="true" />
      </Button>
      <Menu
        v-if="anchor"
        :items="items"
        :at="{ x: anchor.x, y: anchor.y }"
        :viewport="{ width: anchor.x + MARGIN, height: anchor.height }"
        :margin="MARGIN"
        :from="opener"
        :name="words.connectionActions"
        trigger="keyboard"
        is-open
        @choose="onChoose"
        @dismiss="onClose"
      />
    </div>
  </li>
</template>

<style scoped>
.connection-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
  padding: 0.625rem 0.875rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  list-style: none;
}

.connection-row:last-child {
  border-block-end: none;
}

.connection-row__main {
  min-inline-size: 0;
}

.connection-row__title {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: baseline;
}

.connection-row__name {
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
  gap: 0.5rem;
  align-items: center;
  justify-content: flex-end;
}

.connection-row__chip {
  padding: 0.0625rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-pill);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
}

.connection-row__chip--bad {
  border-color: var(--numen-alarm);
  color: var(--numen-alarm);
}

.connection-row__chip--warn {
  color: var(--numen-ink);
}

.connection-row__menu {
  border-radius: var(--numen-radius);
}
</style>
