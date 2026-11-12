<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * One settings row: the title and an info tooltip on the left, the control on
 * the right. When the nearest size container (the settings column) is narrower
 * than 34rem, the control moves under the title at full width.
 *
 * The title carries the id the control is labelled by, because switches,
 * segments and composite controls have no element a `<label for>` can target.
 *
 * A marked row is the setting a health item of its plugin names; a focused row is scrolled into view
 * when it is drawn and whenever it becomes focused.
 */
import { computed, onMounted, ref, watch } from 'vue'
import SettingInfo from './SettingInfo.vue'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  isMarked?: boolean
  isFocused?: boolean
}>()

defineSlots<{
  /** The control, given the id of the title that labels it. */
  default(props: { labelledBy: string }): VNode[]
  /** An action placed after the title and its info tooltip. */
  action?(): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const nameId = computed(() => `settings-${props.at}`)
const trimmedDetail = computed(() => props.detail.trim())
const row = ref<HTMLElement | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (props.isFocused) scrollIntoView()
})

watch(
  () => props.isFocused,
  (isFocused) => {
    if (isFocused) scrollIntoView()
  },
)

/* -------------------------------- Helpers --------------------------------- */
function scrollIntoView(): void {
  row.value?.scrollIntoView?.({ block: 'center' })
}
</script>

<template>
  <div
    ref="row"
    class="settings__row"
    :class="{ 'settings__row--marked': props.isMarked }"
    :data-marked="props.isMarked ? '' : undefined"
  >
    <div class="settings__label">
      <span class="settings__title">
        <span :id="nameId" class="settings__name">{{ name }}</span>
        <SettingInfo
          v-if="trimmedDetail"
          :at="`${nameId}-more`"
          :label="words.aboutSetting(name)"
          :text="trimmedDetail"
        />
        <slot name="action" />
      </span>
    </div>
    <div class="settings__value">
      <slot v-bind="{ labelledBy: nameId }" />
    </div>
  </div>
</template>

<style scoped>
.settings__row {
  --aside-width: 24rem;
  --gap: 0.5rem;
  --pad-block: 0.625rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) fit-content(var(--aside-width));
  align-items: center;
  gap: var(--gap) var(--numen-panel-gap);
  padding-block: var(--pad-block);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.settings__row--marked {
  --mark-width: 0.1875rem;
  --mark-pad: 0.5rem;
  padding-inline-start: var(--mark-pad);
  border-inline-start: var(--mark-width) solid var(--numen-alarm);
}

.settings__row:last-child {
  border-block-end: none;
}

.settings__label {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
}

.settings__title {
  --gap: 0.375rem;
  --min-height: 2rem;
  display: inline-flex;
  align-items: center;
  gap: var(--gap);
  min-block-size: var(--min-height);
  min-inline-size: 0;
}

.settings__name {
  min-inline-size: 0;
  overflow-wrap: anywhere;
  color: var(--numen-ink);
  font-weight: 500;
}

.settings__value {
  --min-height: 2rem;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-block-size: var(--min-height);
  min-inline-size: 0;
}

.settings__value:has(.settings__path--multi) {
  display: contents;
}

.settings__value :deep(.settings__choice),
.settings__value :deep(.settings__text-input) {
  inline-size: var(--settings-choice);
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
