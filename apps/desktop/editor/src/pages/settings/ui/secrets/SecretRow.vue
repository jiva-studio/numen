<script setup lang="ts">
/**
 * One secret of a group as a row of four columns: its name, edited in place on click, the value
 * (masked while one is stored), the plugins that may read it, and the controls to move and delete.
 */
import { nextTick, ref, useTemplateRef } from 'vue'
import { ArrowDown, ArrowUp, Trash2 } from '@lucide/vue'
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
  /** Whether the group holds more than one secret, so that the order can change. */
  canMove: boolean
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'move-up'): void
  (e: 'move-down'): void
  (e: 'rename', name: string): void
  (e: 'write', value: string): void
  (e: 'delete', opener: HTMLElement): void
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

function onDelete(event: MouseEvent): void {
  if (event.currentTarget instanceof HTMLElement) emit('delete', event.currentTarget)
}
</script>

<template>
  <li class="secret-row">
    <input
      v-if="isRenaming"
      ref="nameInput"
      v-model="draftName"
      type="text"
      :aria-label="words.secretRenameName"
      class="secret-row__name-input"
      @keydown.enter="onCommitRename"
      @keydown.esc="isRenaming = false"
      @change="onCommitRename"
      @blur="onCommitRename"
    />
    <button
      v-else
      type="button"
      class="secret-row__name"
      :aria-label="words.secretRename"
      :title="words.secretRename"
      @mousedown.prevent
      @click="onStartRename"
    >
      {{ secret.name }}
    </button>

    <SecretValueInput
      :label="words.secretValue(secret.name)"
      :placeholder="secret.hasValue ? words.secretMasked : words.keyPlaceholder"
      @commit="(value) => emit('write', value)"
    />

    <SecretPermissions
      :secret="secret"
      :declarations="declarations"
      @toggle="(declaration, isAllowed) => emit('toggle', declaration, isAllowed)"
    />

    <div class="secret-row__actions">
      <Button
        v-if="canMove"
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
        v-if="canMove"
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
        class="secret-row__delete"
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretDelete"
        :title="words.secretDelete"
        @click="onDelete"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </div>
  </li>
</template>

<style scoped>
.secret-row {
  display: grid;
  grid-template-columns: 8rem minmax(0, 1fr) 9rem 5.5rem;
  align-items: center;
  gap: 0.5rem var(--numen-panel-gap);
  padding-block: 0.5rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  list-style: none;
}

.secret-row__name,
.secret-row__name-input {
  box-sizing: border-box;
  inline-size: 8rem;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secret-row__name {
  overflow: hidden;
  background: none;
  border: var(--numen-stroke) solid transparent;
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-weight: 500;
  text-align: start;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: text;
}

.secret-row__name-input {
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
}

.secret-row__actions {
  display: flex;
  gap: 0.125rem;
  min-inline-size: 1.75rem;
}

.secret-row__delete {
  opacity: 0;
}

.secret-row:hover .secret-row__delete,
.secret-row:focus-within .secret-row__delete {
  opacity: 1;
}

@media (hover: none) {
  .secret-row__delete {
    opacity: 1;
  }
}
</style>
