<script setup lang="ts">
/**
 * One settings row: the title with a one-line description under it on the left, the control on the
 * right. A setting with details shows an info button beside the title that opens them in a popover.
 * Escape closes it and gives the focus back to the button, and a press outside closes it; a scroll or
 * a resize of the page leaves it open, since it moves with its row. A value changed from its default
 * shows a Reset action beside the title. When the settings column is narrower than 34rem, the control
 * moves under the title at full width.
 *
 * The title carries the id the control is labelled by, because switches, segments and composite
 * controls have no element a `<label for>` can target; the description carries the id the
 * control is described by, so its whole text reaches the control where the row cuts it short.
 *
 * A marked row is the setting a health item of its plugin names. A row is scrolled into view when it
 * is drawn with a focus request and whenever it is given a new one, so a repeated request scrolls
 * again.
 */
import { computed, nextTick, onMounted, ref, useTemplateRef, watch, type VNode } from 'vue'
import { Info, RotateCcw, TriangleAlert } from '@lucide/vue'
import { runInBackground, isPageMove, useMenuDismissal } from '@numen/ui'
import { WORDS as words } from '../../words'
import type { SettingRowProps } from './SettingRow.types'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<SettingRowProps>()

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
const trimmedDetails = computed(() => props.details?.trim() ?? '')
const row = useTemplateRef<HTMLElement>('row')
const label = useTemplateRef<HTMLElement>('label')
const infoButton = useTemplateRef<HTMLButtonElement>('infoButton')
const popover = useTemplateRef<HTMLElement>('popover')
const isInfoOpen = ref(false)
const dismissal = useMenuDismissal(label, () => closeInfo(false), isPageMove)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (props.focus) scrollIntoView()
})

watch(isInfoOpen, (isOpen) => {
  if (!isOpen) {
    dismissal.release()
    return
  }
  dismissal.listen()
  runInBackground(
    nextTick(() => popover.value?.focus()),
    () => undefined,
  )
})

watch(
  () => props.focus,
  (focus) => {
    if (focus) scrollIntoView()
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onReset(): void {
  emit('reset')
}

function onToggleInfo(): void {
  isInfoOpen.value = !isInfoOpen.value
}

function onPopoverEscape(): void {
  closeInfo(true)
}

/* -------------------------------- Helpers --------------------------------- */
function closeInfo(isFocusReturned: boolean): void {
  isInfoOpen.value = false
  if (isFocusReturned) infoButton.value?.focus()
}

function scrollIntoView(): void {
  row.value?.scrollIntoView({ block: 'center' })
}
</script>

<template>
  <div ref="row" class="settings__row" :data-marked="props.isMarked ? '' : undefined">
    <div ref="label" class="settings__label">
      <span :id="nameId" class="settings__name">{{ props.name }}</span>
      <button
        v-if="trimmedDetails"
        ref="infoButton"
        type="button"
        class="settings__info"
        :aria-label="words.aboutSetting(props.name)"
        aria-haspopup="dialog"
        :aria-expanded="isInfoOpen"
        @click="onToggleInfo"
      >
        <Info :size="14" aria-hidden="true" />
      </button>
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
      <div
        v-if="isInfoOpen"
        ref="popover"
        class="settings__popover"
        role="dialog"
        :aria-label="words.aboutSetting(props.name)"
        tabindex="-1"
        @keydown.esc.stop.prevent="onPopoverEscape"
      >
        <p class="settings__popover-title">{{ props.name }}</p>
        <p class="settings__popover-text">{{ trimmedDetails }}</p>
      </div>
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

.settings__info {
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

.settings__info:hover,
.settings__info[aria-expanded='true'] {
  background-color: var(--settings-hover-bg);
  color: var(--numen-ink);
}

.settings__info:focus-visible,
.settings__popover:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.settings__popover {
  position: absolute;
  inset-block-start: calc(100% + 0.25rem);
  inset-inline-start: 0;
  z-index: 1;
  box-sizing: border-box;
  inline-size: max-content;
  max-inline-size: min(26rem, 100cqi);
  padding: 0.75rem 0.875rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-popup);
  background-color: var(--numen-raised);
  box-shadow: var(--numen-shadow-card);
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  line-height: 1.5;
}

.settings__popover-title {
  margin: 0 0 0.25rem;
  font-weight: 600;
}

.settings__popover-text {
  margin: 0;
  color: var(--numen-hushed);
}

.settings__mark {
  flex: none;
  color: var(--numen-alarm);
}

.settings__row:last-child {
  border-block-end: none;
}

.settings__label {
  position: relative;
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
