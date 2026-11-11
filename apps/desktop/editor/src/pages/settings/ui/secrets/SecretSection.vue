<script setup lang="ts">
/**
 * The secrets of one group in priority order. A group with no secret shows one empty secret
 * while it has none, named after the service of the group, else Default. The rows show their
 * names, under a Name label, only while the group holds more than one secret. Adding a secret
 * puts the focus into the new row, and discarding one returns it to the button that adds.
 */
import { computed, useId, useTemplateRef } from 'vue'
import { getFirstSecretName, getMoveTarget, type Secret } from '@/entities/settings'
import {
  getSectionDeclarations,
  type Declaration,
  type SecretSection,
} from '../../lib/secretSections'
import type { SecretDraft } from '../../model/secretDrafts'
import { runAfterRender } from '@/shared/model/afterRender'
import SecretColumnLabels from './SecretColumnLabels.vue'
import SecretDraftRow from './SecretDraftRow.vue'
import SecretRow from './SecretRow.vue'
import SecretSectionTitle from './SecretSectionTitle.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  section: SecretSection
  groups: readonly string[]
  drafts: readonly SecretDraft[]
  declarations: readonly Declaration[]
  isNew?: boolean
  create: (name: string, value: string, draftId?: number) => Promise<boolean>
}>()

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'rename-group', name: string): void
  (e: 'delete-group', opener: HTMLElement): void
  (e: 'discard', id: number): void
  (e: 'move', id: string, beforeId: string): void
  (e: 'rename', id: string, name: string): void
  (e: 'write', id: string, value: string): void
  (e: 'delete', secret: Secret, opener: HTMLElement): void
  (e: 'toggle', secret: Secret, declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const headingId = useId()
const title = useTemplateRef<InstanceType<typeof SecretSectionTitle>>('title')
const firstName = computed(() => getFirstSecretName(props.section.group))
const isFirstEmpty = computed(() => props.section.secrets.length === 0)
const keyCount = computed(
  () => props.section.secrets.length + props.drafts.length + (isFirstEmpty.value ? 1 : 0),
)
const hasNames = computed(() => keyCount.value > 1)

/* -------------------------------- Handlers -------------------------------- */
function onMove(index: number, direction: 'up' | 'down'): void {
  const target = getMoveTarget(props.section.secrets, index, direction)
  if (target) emit('move', target.id, target.beforeId)
}

function onDiscard(id: number): void {
  emit('discard', id)
  runAfterRender(() => title.value?.focusAdd())
}
</script>

<template>
  <section class="secret-section" :aria-labelledby="headingId">
    <SecretSectionTitle
      ref="title"
      :group="section.group"
      :groups="groups"
      :heading-id="headingId"
      :is-asked="section.isAsked"
      :is-new="isNew ?? false"
      @add="emit('add')"
      @rename="(name) => emit('rename-group', name)"
      @delete-group="(opener) => emit('delete-group', opener)"
    />

    <ul class="secret-section__secrets">
      <SecretColumnLabels v-if="keyCount > 0" :has-name="hasNames" />
      <SecretDraftRow
        v-if="isFirstEmpty"
        :name="firstName"
        :has-name="hasNames"
        :can-discard="false"
        :submit="create"
      />
      <SecretRow
        v-for="(secret, index) in section.secrets"
        :key="secret.id"
        :secret="secret"
        :is-first="index === 0"
        :is-last="index === section.secrets.length - 1"
        :can-move="section.secrets.length > 1"
        :has-name="hasNames"
        :declarations="getSectionDeclarations(declarations, section.group)"
        @move-up="onMove(index, 'up')"
        @move-down="onMove(index, 'down')"
        @rename="(name) => emit('rename', secret.id, name)"
        @write="(value) => emit('write', secret.id, value)"
        @delete="(opener) => emit('delete', secret, opener)"
        @toggle="(declaration, isAllowed) => emit('toggle', secret, declaration, isAllowed)"
      />
      <SecretDraftRow
        v-for="draft in drafts"
        :key="draft.id"
        :name="draft.name"
        :has-name="hasNames"
        can-discard
        has-autofocus
        :submit="(name, value) => create(name, value, draft.id)"
        @discard="onDiscard(draft.id)"
      />
    </ul>
  </section>
</template>

<style scoped>
.secret-section {
  --pad: 0.875rem 1rem;
  display: flex;
  flex-direction: column;
  margin-block-end: calc(var(--settings-apart) / 2);
  padding: var(--pad);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.secret-section__secrets {
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
