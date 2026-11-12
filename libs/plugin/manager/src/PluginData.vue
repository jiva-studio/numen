<script setup lang="ts">
/**
 * The folder a plugin keeps in the shown vault: how many files it holds, and the deletion, which
 * the person confirms before it is asked for.
 */
import { ref } from 'vue'
import { Button } from '@numen/ui'
import DetailRow from './DetailRow.vue'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  pluginId: string
  pluginName: string
  fileCount: number
  problem: string
}>()

const emit = defineEmits<{
  (e: 'delete-data'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isAsking = ref(false)

/* -------------------------------- Handlers -------------------------------- */
function onDelete(): void {
  isAsking.value = true
}

function onKeep(): void {
  isAsking.value = false
}

function onConfirm(): void {
  isAsking.value = false
  emit('delete-data')
}
</script>

<template>
  <div class="plugin-data">
    <div v-if="props.fileCount > 0" class="plugin-data__card">
      <DetailRow
        :at="`data-${props.pluginId}`"
        :name="words.pluginData"
        :detail="words.pluginDataDetail(props.fileCount)"
      >
        <Button variant="danger" size="small" @click="onDelete">
          {{ words.pluginDataDelete }}
        </Button>
      </DetailRow>
    </div>
    <div
      v-if="isAsking"
      class="plugin-data__asking"
      role="alert"
      :aria-label="words.pluginDataDelete"
    >
      <span>{{ words.pluginDataAsking(props.pluginName, props.fileCount) }}</span>
      <span class="plugin-data__answers">
        <Button variant="danger" size="small" @click="onConfirm">
          {{ words.pluginDataConfirm }}
        </Button>
        <Button variant="outline" size="small" @click="onKeep">
          {{ words.pluginDataKeep }}
        </Button>
      </span>
    </div>
    <p v-if="props.problem" class="plugin-data__problem" role="alert">{{ props.problem }}</p>
  </div>
</template>

<style scoped>
.plugin-data {
  --data-pad-inline: 1rem;
  --asking-gap: 0.375rem;
  --asking-pad-block: 0.5rem;
  --asking-pad-inline: 0.75rem;
}

.plugin-data__card {
  padding-inline: var(--data-pad-inline);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-data__asking {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--asking-gap);
  margin-block-start: var(--asking-gap);
  padding: var(--asking-pad-block) var(--asking-pad-inline);
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius-tight);
  color: var(--numen-ink);
}

.plugin-data__answers {
  display: flex;
  gap: var(--asking-gap);
}

.plugin-data__problem {
  margin: var(--asking-gap) 0 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
