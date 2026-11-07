<script setup lang="ts">
/**
 * One secret of a group: its name, the controls to move, rename and delete it, a masked field for
 * a new value, and a check box for each declaration of a plugin that may be allowed to read it.
 */
import { nextTick, ref, useTemplateRef } from 'vue'
import { ArrowDown, ArrowUp, Pencil, Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import type { Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SecretPermissions from './SecretPermissions.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret: Secret
  isFirst: boolean
  isLast: boolean
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'move-up'): void
  (e: 'move-down'): void
  (e: 'rename', name: string): void
  (e: 'write', value: string): void
  (e: 'delete'): void
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const isRenaming = ref(false)
const draftName = ref('')
const nameInput = useTemplateRef<HTMLInputElement>('nameInput')

/* -------------------------------- Handlers -------------------------------- */
async function onStartRename(): Promise<void> {
  draftName.value = props.secret.name
  isRenaming.value = true
  await nextTick()
  nameInput.value?.select()
}

function onCommitRename(): void {
  if (!isRenaming.value) return
  isRenaming.value = false
  const name = draftName.value.trim()
  if (name !== '' && name !== props.secret.name) emit('rename', name)
}

function onCancelRename(): void {
  isRenaming.value = false
}
</script>

<template>
  <li class="secret-row">
    <div class="secret-row__title">
      <input
        v-if="isRenaming"
        ref="nameInput"
        v-model="draftName"
        type="text"
        :aria-label="words.secretRenameName"
        class="secret-row__name-input"
        @keydown.enter="onCommitRename"
        @keydown.esc="onCancelRename"
        @change="onCommitRename"
        @blur="onCommitRename"
      />
      <span v-else class="secret-row__name">{{ secret.name }}</span>
      <span v-if="!secret.hasValue" class="secret-row__missing">{{ words.secretNoValue }}</span>
    </div>
    <div class="secret-row__actions">
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretMoveUp"
        :title="words.secretMoveUp"
        :disabled="isFirst"
        @click="emit('move-up')"
      >
        <ArrowUp :size="14" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretMoveDown"
        :title="words.secretMoveDown"
        :disabled="isLast"
        @click="emit('move-down')"
      >
        <ArrowDown :size="14" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretRename"
        :title="words.secretRename"
        @mousedown.prevent
        @click="onStartRename"
      >
        <Pencil :size="14" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretDelete"
        :title="words.secretDelete"
        @click="emit('delete')"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </div>

    <SecretValueInput
      class="secret-row__value"
      :label="words.secretValue(secret.name)"
      :placeholder="secret.hasValue ? words.secretStored : words.keyPlaceholder"
      @commit="(value) => emit('write', value)"
    />

    <SecretPermissions
      class="secret-row__allowed"
      :secret="secret"
      :declarations="declarations"
      @toggle="(declaration, isAllowed) => emit('toggle', declaration, isAllowed)"
    />
  </li>
</template>

<style scoped>
.secret-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.5rem var(--numen-panel-gap);
  padding-block: 0.75rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  list-style: none;
}

.secret-row__title {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  min-inline-size: 0;
}

.secret-row__name {
  color: var(--numen-ink);
  font-weight: 500;
  overflow-wrap: anywhere;
}

.secret-row__missing {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.secret-row__name-input {
  box-sizing: border-box;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secret-row__actions {
  display: flex;
  flex-shrink: 0;
  gap: 0.125rem;
}

.secret-row__value,
.secret-row__allowed {
  grid-column: 1 / -1;
}
</style>
