<script setup lang="ts">
import { computed } from 'vue'
import type { NameEntryState } from '../../model/nameEntry'
import type { Objection } from '../../lib/order'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** Name entry state managing this input field. */
  naming: NameEntryState<Objection>
  /** Target identifier addressed by the name entry state. */
  targetId: string
  /** Placeholder and accessible label for the input element. */
  stem: string
  /** Validation description identifier for aria-describedby. */
  describedBy?: string | null
}>()

/* --------------------------------- State ---------------------------------- */
const isInvalid = computed<boolean>(() => props.naming.getObjection(props.targetId) !== null)

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  const target = event.target as HTMLInputElement
  props.naming.setDraft(props.targetId, target.value)
}

function onChange(): void {
  props.naming.commit(props.targetId)
}

function onKeydown(event: KeyboardEvent): void {
  props.naming.onKey(event, props.targetId)
}
</script>

<template>
  <input
    class="name-box"
    type="text"
    size="1"
    :value="naming.getText(targetId)"
    :placeholder="stem"
    :aria-label="stem"
    :aria-invalid="isInvalid || undefined"
    :aria-describedby="describedBy ?? undefined"
    @input="onInput"
    @change="onChange"
    @keydown="onKeydown"
  />
</template>

<style scoped>
/* Input padding when rendered inside a card row container. */
.name-box {
  padding: var(--card-row-pad-block) var(--card-row-pad-inline);
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  cursor: auto;
}

.name-box:focus-visible {
  outline: none;
}
</style>
