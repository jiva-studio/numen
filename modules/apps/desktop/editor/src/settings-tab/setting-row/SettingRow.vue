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

const labelling = computed(() => `settings-${props.at}`)
</script>

<template>
  <div class="settings__row">
    <span class="settings__said">
      <span :id="labelling" class="settings__name">{{ name }}</span>
      <span class="settings__detail">{{ detail }}</span>
    </span>
    <span class="settings__value">
      <slot v-bind="{ labelledBy: labelling }" />
    </span>
  </div>
</template>

<style scoped>
/* One row: the air around it, and the space between what it is called and what
   it means. */
.settings__row {
  --settings-row-air: 0.5rem;
  --settings-said-gap: 0.125rem;
  /* The name and what it means on the left, the control at the end of the row. */
  display: grid;
  grid-template-columns: 1fr max-content;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

/* What the row is called, and under it what it means. */
.settings__said {
  display: flex;
  flex-direction: column;
  gap: var(--settings-said-gap);
  min-inline-size: 0;
}

/* The name of a row and what it means are one size, and the name carries the
   weight and the colour that tell them apart. */
.settings__name {
  color: var(--numen-ink);
  font-weight: 500;
}

/* Controls of every width end at the one edge. */
.settings__value {
  display: flex;
  align-items: center;
  justify-content: end;
}

.settings__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
