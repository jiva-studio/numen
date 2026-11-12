<script setup lang="ts">
/**
 * One settings row: the title with a one-line description under it on the left, the control on the
 * right. A value changed from its default shows a Reset action beside the title. When the settings
 * column is narrower than 34rem, the control moves under the title at full width.
 *
 * The title carries the id the control is labelled by, because switches, segments and composite
 * controls have no element a `<label for>` can target; the description carries the id the
 * control is described by, so its whole text reaches the control where the row cuts it short.
 *
 * A marked row is the setting a health item of its plugin names; a focused row is scrolled into view
 * when it is drawn and whenever it becomes focused.
 */
import { computed, onMounted, useTemplateRef, watch, type VNode } from 'vue'
import { RotateCcw, TriangleAlert } from '@lucide/vue'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  isMarked?: boolean
  isFocused?: boolean
  isChanged?: boolean
}>()

const emit = defineEmits<{
  (e: 'reset'): void
}>()

defineSlots<{
  /** The control, given the ids of the title that labels it and of the description, if any. */
  default(props: { labelledBy: string; describedBy: string | undefined }): VNode[]
  /** An action placed after the title. */
  action?(): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const nameId = computed(() => `settings-${props.at}`)
const trimmedDetail = computed(() => props.detail.trim())
const detailId = computed(() => (trimmedDetail.value ? `${nameId.value}-detail` : undefined))
const row = useTemplateRef<HTMLElement>('row')

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

/* -------------------------------- Handlers -------------------------------- */
function onReset(): void {
  emit('reset')
}

/* -------------------------------- Helpers --------------------------------- */
function scrollIntoView(): void {
  row.value?.scrollIntoView({ block: 'center' })
}
</script>

<template>
  <div
    ref="row"
    class="settings__row"
    :data-marked="props.isMarked ? '' : undefined"
  >
    <div class="settings__label">
      <span :id="nameId" class="settings__name">{{ props.name }}</span>
      <TriangleAlert
        v-if="props.isMarked"
        class="settings__mark"
        :size="14"
        :stroke-width="1.75"
        aria-hidden="true"
      />
      <button
        v-if="props.isChanged"
        type="button"
        class="settings__reset"
        :aria-label="words.resetSettingOf(props.name)"
        :title="words.resetSettingOf(props.name)"
        @click="onReset"
      >
        <RotateCcw :size="14" aria-hidden="true" />
      </button>
      <slot name="action" />
      <span v-if="trimmedDetail" :id="detailId" class="settings__detail" :title="trimmedDetail">
        {{ trimmedDetail }}
      </span>
    </div>
    <div class="settings__value">
      <slot v-bind="{ labelledBy: nameId, describedBy: detailId }" />
    </div>
  </div>
</template>

<style scoped>
.settings__row {
  --gap: 0.5rem;
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--settings-control);
  align-items: center;
  gap: var(--gap) var(--numen-panel-gap);
  box-sizing: border-box;
  min-block-size: var(--settings-row-min);
  padding-block: var(--settings-row-pad);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.settings__mark {
  flex: none;
  color: var(--numen-alarm);
}

.settings__row:last-child {
  border-block-end: none;
}

.settings__label {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.125rem 0.375rem;
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
  background-color: var(--settings-hover-bg);
  color: var(--numen-ink);
}

.settings__detail {
  flex-basis: 100%;
  overflow: hidden;
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

@container settings (max-width: 34rem) {
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
