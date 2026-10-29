<script setup lang="ts">
/**
 * The path held by a plugin setting of type file or folder: its last parts as the label with the full
 * path on hover, the platform file picker to choose another path, and a clear action. The reason a
 * chosen path was rejected is shown below it.
 */
import { computed } from 'vue'
import { Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  path: string
  /** Why the setting did not take the path last chosen; empty where it did. */
  problem: string
  labelledBy: string
  isMulti?: boolean
  words: {
    readonly pathNone: string
    readonly pathChoose: string
    readonly pathAdd?: string
    readonly pathClear: string
  }
}>()

const emit = defineEmits<{
  choose: []
  clear: [targetPath?: string]
}>()

/* --------------------------------- State ---------------------------------- */
const paths = computed(() => parsePaths(props.path))
const hasMultipleRows = computed(() => Boolean(props.isMulti))
const multiRowPaths = computed(() => (hasMultipleRows.value ? paths.value : []))
const chooseButtonLabel = computed(() => getChooseButtonLabel(props.isMulti, props.words))

/* -------------------------------- Handlers -------------------------------- */
function onChoose() {
  emit('choose')
}

function onClear(targetPath?: string) {
  emit('clear', targetPath)
}

/* -------------------------------- Helpers --------------------------------- */
function parsePaths(rawPaths: string): string[] {
  return rawPaths
    .split('\n')
    .map((one) => one.trim())
    .filter((one) => one !== '')
}

function getChooseButtonLabel(
  isMulti: boolean | undefined,
  words: { readonly pathChoose: string; readonly pathAdd?: string },
): string {
  if (isMulti) {
    return words.pathAdd ?? words.pathChoose
  }
  return words.pathChoose
}

function formatPathLabel(rawPath: string): string {
  const parts = rawPath.split(/[\\/]/).filter((one) => one !== '')
  return parts.length > 2 ? `…/${parts.slice(-2).join('/')}` : rawPath
}
</script>

<template>
  <span
    class="settings__path"
    :class="{ 'settings__path--multi': hasMultipleRows }"
    role="group"
    :aria-labelledby="labelledBy"
  >
    <span v-if="!hasMultipleRows" class="settings__path-line">
      <span
        v-if="paths[0]"
        class="settings__path-label"
        :title="paths[0]"
        data-testid="setting-path"
      >
        {{ formatPathLabel(paths[0]) }}
      </span>
      <Button variant="outline" size="small" class="settings__path-choose" @click="onChoose">
        {{ chooseButtonLabel }}
      </Button>
      <Button
        v-if="paths[0]"
        variant="ghost"
        size="icon-small"
        class="settings__path-clear"
        :aria-label="words.pathClear"
        :title="words.pathClear"
        @click="onClear(paths[0])"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </span>
    <span v-else class="settings__path-header-action">
      <Button variant="outline" size="small" class="settings__path-choose" @click="onChoose">
        {{ chooseButtonLabel }}
      </Button>
    </span>
    <span v-for="item in multiRowPaths" :key="item" class="settings__path-item">
      <span
        class="settings__path-label settings__path-label--full"
        :title="item"
        data-testid="setting-path"
      >
        {{ item }}
      </span>
      <Button
        variant="ghost"
        size="icon-small"
        class="settings__path-clear"
        :aria-label="words.pathClear"
        :title="words.pathClear"
        @click="onClear(item)"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </span>
    <span v-if="problem !== ''" class="settings__path-problem" role="alert">{{ problem }}</span>
  </span>
</template>

<style scoped>
.settings__path {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.375rem;
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__path--multi {
  display: contents;
}

.settings__path-header-action {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  justify-self: end;
}

.settings__path-item {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-block-size: 1.875rem;
  min-inline-size: 0;
}

.settings__path-header-action + .settings__path-item {
  padding-block-start: 0.375rem;
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
}

.settings__path-line {
  display: flex;
  align-items: center;
  gap: var(--settings-near);
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__path-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-inline-size: 18rem;
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
}

.settings__path-label--full {
  max-inline-size: 100%;
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.settings__path-clear {
  color: var(--numen-hushed);
}

.settings__path-clear:hover {
  color: var(--numen-alarm);
}

.settings__path-choose {
  background-color: var(--numen-field-bg);
  border-color: var(--numen-field-rule);
}

.settings__path-problem {
  grid-column: 1 / -1;
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
