<script setup lang="ts">
/**
 * One settings row: the title and a one-sentence description on the left, the
 * control on the right. Text after the first sentence of `detail` goes into a
 * tooltip behind an info icon next to the title. When the nearest size
 * container (the settings column) is narrower than 34rem, the control moves
 * under the description at full width.
 *
 * The title carries the id the control is labelled by, because switches,
 * segments and composite controls have no element a `<label for>` can target.
 */
import { computed } from 'vue'
import SettingInfo from './SettingInfo.vue'
import { splitDetail } from './detail'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** Unique key of the setting; the title id is derived from it. */
  at: string
  name: string
  detail: string
}>()

defineSlots<{
  /** The control, given the id of the title that labels it. */
  default(props: { labelledBy: string }): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const nameId = computed(() => `settings-${props.at}`)
const parts = computed(() => splitDetail(props.detail))
</script>

<template>
  <div class="settings__row">
    <div class="settings__label">
      <span class="settings__title">
        <span :id="nameId" class="settings__name">{{ name }}</span>
        <SettingInfo
          v-if="parts.more"
          :at="`${nameId}-more`"
          :label="words.aboutSetting(name)"
          :text="parts.more"
        />
      </span>
      <span v-if="parts.summary" class="settings__detail">{{ parts.summary }}</span>
    </div>
    <div class="settings__value">
      <slot v-bind="{ labelledBy: nameId }" />
    </div>
  </div>
</template>

<style scoped>
.settings__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) fit-content(24rem);
  align-items: start;
  gap: 0.5rem var(--numen-panel-gap, 1rem);
  padding-block: 0.625rem;
  border-block-end: var(--numen-stroke, 1px) solid var(--numen-rule);
}

.settings__row:last-child {
  border-block-end: none;
}

.settings__label {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-inline-size: 0;
}

.settings__title {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  min-block-size: 2rem;
  min-inline-size: 0;
}

.settings__name {
  min-inline-size: 0;
  overflow-wrap: anywhere;
  color: var(--numen-ink);
  font-weight: 500;
}

.settings__detail {
  overflow-wrap: anywhere;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.settings__value {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-block-size: 2rem;
  min-inline-size: 0;
}

.settings__value :deep(.settings__choice),
.settings__value :deep(.settings__text-input) {
  inline-size: var(--settings-choice, 16rem);
  max-inline-size: 100%;
}

@container (max-width: 34rem) {
  .settings__row {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings__value {
    --settings-choice: 100%;
    justify-content: flex-start;
  }

  .settings__value > :deep(*) {
    max-inline-size: 100%;
  }
}
</style>
