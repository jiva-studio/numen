<script setup lang="ts">
/**
 * One row of the plugin page: the title with its explanation on the left, the control on the
 * right. The title carries the id the control is labelled by, because a switch has no element a
 * `<label for>` can target.
 */
import { computed } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** Unique key of the row; the title id is derived from it. */
  at: string
  name: string
  detail: string
}>()

defineSlots<{
  /** The control, given the id of the title that labels it. */
  default(props: { labelledBy: string }): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const nameId = computed(() => `plugin-row-${props.at}`)
</script>

<template>
  <div class="detail-row">
    <div class="detail-row__label">
      <span :id="nameId" class="detail-row__name">{{ props.name }}</span>
      <span v-if="props.detail" class="detail-row__detail">{{ props.detail }}</span>
    </div>
    <div class="detail-row__value">
      <slot v-bind="{ labelledBy: nameId }" />
    </div>
  </div>
</template>

<style scoped>
.detail-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.5rem var(--numen-panel-gap);
  padding-block: 0.625rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.detail-row:last-child {
  border-block-end: none;
}

.detail-row__label {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.detail-row__name {
  overflow-wrap: anywhere;
  color: var(--numen-ink);
  font-weight: 500;
}

.detail-row__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.detail-row__value {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-block-size: 2rem;
}
</style>
