<script setup lang="ts">
/**
 * One setting on a row: what it is called, what it means, and the control that
 * turns it at the end of the line.
 *
 * The name carries the identifier and the control is announced by it. That is
 * the one rule here, and it holds whatever the control is: a switch, a row of
 * segments and a week of days are each drawn out of parts, and none of them has
 * an element a `<label for>` can name.
 */
import { computed } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** What the setting is about, which the name is addressed by. */
  at: string
  /** What the row is called, and under it what it means. */
  name: string
  detail: string
}>()

defineSlots<{
  /** The control at the end of the line, told what announces it. */
  default(props: { labelledBy: string }): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const nameId = computed(() => `settings-${props.at}`)
</script>

<template>
  <div class="settings__row">
    <span class="settings__label">
      <span :id="nameId" class="settings__name">{{ name }}</span>
      <span class="settings__detail">{{ detail }}</span>
    </span>
    <span class="settings__value">
      <slot v-bind="{ labelledBy: nameId }" />
    </span>
  </div>
</template>

<style scoped>
.settings__row {
  --settings-row-air: 0.625rem;
  --settings-label-gap: 0.125rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-row-air) var(--numen-panel-gap, 1rem);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke, 1px) solid var(--numen-rule);
}

.settings__row:last-child {
  border-block-end: none;
}

/* The name keeps its width beside a control of any width, and the control gives way. */
.settings__label {
  display: flex;
  flex-direction: column;
  gap: var(--settings-label-gap);
  flex: 1 1 auto;
  min-inline-size: min(100%, 18rem);
}

.settings__name {
  color: var(--numen-ink);
  font-weight: 500;
}

.settings__value {
  display: flex;
  flex: 0 1 auto;
  min-inline-size: 0;
  align-items: center;
  justify-content: flex-end;
}

.settings__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
