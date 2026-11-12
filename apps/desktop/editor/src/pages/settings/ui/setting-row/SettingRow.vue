<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * One settings row: the title with its description under it on the left, the control on the
 * right. A value changed from its default shows a Reset action beside the title. When the nearest
 * size container (the settings column) is narrower than 34rem, the control moves under the title
 * at full width.
 *
 * The title carries the id the control is labelled by, because switches,
 * segments and composite controls have no element a `<label for>` can target.
 *
 * A marked row is the setting a health item of its plugin names; a focused row is scrolled into view
 * when it is drawn and whenever it becomes focused.
 */
import { computed, onMounted, ref, watch } from 'vue'
import { RotateCcw } from '@lucide/vue'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  isMarked?: boolean
  isFocused?: boolean
  isChanged?: boolean
  isDependent?: boolean
}>()

const emit = defineEmits<{
  (e: 'reset'): void
}>()

defineSlots<{
  /** The control, given the id of the title that labels it. */
  default(props: { labelledBy: string }): VNode[]
  /** An action placed after the title. */
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

/* -------------------------------- Handlers -------------------------------- */
function onReset(): void {
  emit('reset')
}
</script>

<template>
  <div
    ref="row"
    class="settings__row"
    :class="{
      'settings__row--marked': props.isMarked,
      'settings__row--dependent': props.isDependent,
    }"
    :data-marked="props.isMarked ? '' : undefined"
  >
    <div class="settings__label">
      <span class="settings__title">
        <span :id="nameId" class="settings__name">{{ name }}</span>
        <button
          v-if="props.isChanged"
          type="button"
          class="settings__reset"
          :aria-label="words.resetSettingOf(name)"
          :title="words.resetSettingOf(name)"
          @click="onReset"
        >
          <RotateCcw :size="14" aria-hidden="true" />
        </button>
        <slot name="action" />
      </span>
      <span
        v-if="trimmedDetail"
        :id="`${nameId}-more`"
        class="settings__detail"
        :title="trimmedDetail"
      >
        {{ trimmedDetail }}
      </span>
    </div>
    <div class="settings__value">
      <slot v-bind="{ labelledBy: nameId }" />
    </div>
  </div>
</template>

<style scoped>
.settings__row {
  --gap: 0.5rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--settings-control, 16.25rem);
  align-items: center;
  gap: var(--gap) var(--numen-panel-gap);
  box-sizing: border-box;
  min-block-size: var(--settings-row-min, 3.25rem);
  padding-block: var(--settings-row-pad, 0.625rem);
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
  display: inline-flex;
  align-items: center;
  gap: var(--gap);
  min-inline-size: 0;
}

.settings__name {
  min-inline-size: 0;
  overflow-wrap: anywhere;
  color: var(--numen-ink);
  font-weight: 500;
}

.settings__reset {
  display: inline-grid;
  place-items: center;
  inline-size: 1.5rem;
  block-size: 1.5rem;
  margin-block: -0.25rem;
  padding: 0;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.settings__reset:hover {
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 8%);
  color: var(--numen-ink);
}

.settings__detail {
  overflow: hidden;
  margin-block-start: 0.125rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  text-overflow: ellipsis;
  white-space: nowrap;
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
.settings__value :deep(.settings__text-input),
.settings__value :deep(.settings__number),
.settings__value :deep(.settings__enum) {
  inline-size: 100%;
}

.settings__value :deep(.settings__enum) {
  align-items: stretch;
}

@container (max-width: 34rem) {
  .settings__row {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings__value {
    justify-content: flex-start;
  }

  .settings__value > :deep(*) {
    max-inline-size: 100%;
  }
}
</style>
