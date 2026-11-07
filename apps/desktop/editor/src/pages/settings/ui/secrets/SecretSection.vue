<script setup lang="ts">
/**
 * The secrets of one group in priority order. A group a plugin asks for shows one empty secret
 * named Default while it has none. A group the person made can be renamed.
 */
import { nextTick, ref, useId, useTemplateRef } from 'vue'
import { Pencil, Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import type { Declaration, SecretSection } from '../../lib/secretSections'
import { trimGroupName } from '../../lib/trimGroupName'
import { WORDS as words } from '../../words'
import SecretDraftRow from './SecretDraftRow.vue'
import SecretRow from './SecretRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  section: SecretSection
  /** The names of the secrets of the group not created yet. */
  drafts: readonly string[]
  declarations: readonly Declaration[]
  /** Creates the secret of a draft and resolves true when it was created; the field keeps a failed value. */
  create: (name: string, value: string) => Promise<boolean>
}>()

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'rename-group', name: string): void
  (e: 'discard', name: string): void
  (e: 'move', id: string, beforeId: string): void
  (e: 'rename', id: string, name: string): void
  (e: 'write', id: string, value: string): void
  (e: 'delete', secret: Secret, opener: HTMLElement): void
  (e: 'toggle', secret: Secret, declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const isRenaming = ref(false)
const draftName = ref('')
const nameInput = useTemplateRef<HTMLInputElement>('nameInput')
const headingId = useId()

/* -------------------------------- Handlers -------------------------------- */
async function onStartRename(): Promise<void> {
  draftName.value = props.section.group
  isRenaming.value = true
  await nextTick()
  nameInput.value?.select()
}

function onCommitRename(): void {
  if (!isRenaming.value) return
  isRenaming.value = false
  const name = trimGroupName(draftName.value)
  if (name !== '' && name !== props.section.group) emit('rename-group', name)
}

function onCancelRename(): void {
  isRenaming.value = false
}

function onMoveUp(index: number): void {
  const above = props.section.secrets[index - 1]
  const secret = props.section.secrets[index]
  if (above && secret) emit('move', secret.id, above.id)
}

function onMoveDown(index: number): void {
  const secret = props.section.secrets[index]
  if (secret) emit('move', secret.id, props.section.secrets[index + 2]?.id ?? '')
}
</script>

<template>
  <section class="secret-section" :aria-labelledby="headingId">
    <div class="secret-section__header">
      <input
        v-if="isRenaming"
        ref="nameInput"
        v-model="draftName"
        type="text"
        :aria-label="words.secretGroupRenameName"
        class="secret-section__name-input"
        @keydown.enter="onCommitRename"
        @keydown.esc="onCancelRename"
        @change="onCommitRename"
        @blur="onCommitRename"
      />
      <h2 v-show="!isRenaming" :id="headingId" class="secret-section__name">{{ section.group }}</h2>
      <Button
        v-if="!section.isAsked && !isRenaming"
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretGroupRename"
        :title="words.secretGroupRename"
        @mousedown.prevent
        @click="onStartRename"
      >
        <Pencil :size="14" aria-hidden="true" />
      </Button>
    </div>

    <ul class="secret-section__secrets">
      <SecretDraftRow
        v-if="section.isAsked && section.secrets.length === 0"
        :name="words.secretDefault"
        :can-discard="false"
        :submit="(value) => create(words.secretDefault, value)"
      />
      <SecretRow
        v-for="(secret, index) in section.secrets"
        :key="secret.id"
        :secret="secret"
        :is-first="index === 0"
        :is-last="index === section.secrets.length - 1"
        :declarations="declarations"
        @move-up="onMoveUp(index)"
        @move-down="onMoveDown(index)"
        @rename="(name) => emit('rename', secret.id, name)"
        @write="(value) => emit('write', secret.id, value)"
        @delete="(opener) => emit('delete', secret, opener)"
        @toggle="(declaration, isAllowed) => emit('toggle', secret, declaration, isAllowed)"
      />
      <SecretDraftRow
        v-for="name in drafts"
        :key="name"
        :name="name"
        can-discard
        :submit="(value) => create(name, value)"
        @discard="emit('discard', name)"
      />
    </ul>

    <Button variant="ghost" size="small" class="secret-section__add" @click="emit('add')">
      <Plus :size="14" aria-hidden="true" />
      {{ words.secretAdd }}
    </Button>
  </section>
</template>

<style scoped>
.secret-section {
  display: flex;
  flex-direction: column;
  margin-block-end: var(--settings-apart);
}

.secret-section__header {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding-block-end: 0.375rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.secret-section__name {
  margin: 0;
  color: var(--numen-ink);
  font-size: var(--numen-text-3);
  font-weight: 600;
}

.secret-section__name-input {
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

.secret-section__secrets {
  margin: 0;
  padding: 0;
  list-style: none;
}

.secret-section__add {
  align-self: flex-start;
  margin-block-start: 0.5rem;
  margin-inline-start: -0.5rem;
}
</style>
