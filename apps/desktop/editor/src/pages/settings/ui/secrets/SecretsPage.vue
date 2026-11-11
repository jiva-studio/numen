<script setup lang="ts">
/**
 * The page of secrets: every secret by group, in priority order, with the plugin declarations that
 * may read it. A value is entered here once and is never shown again.
 */
import { useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { PluginInfo, Secret } from '@/entities/settings'
import type { Declaration, SecretSection as SectionData } from '../../lib/secretSections'
import { runAfterRender } from '@/shared/model/afterRender'
import { useSecretsPage } from '../../model/useSecretsPage'
import type { InstallationSecrets } from '../../types'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import SecretSection from './SecretSection.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secrets: InstallationSecrets
  plugins: readonly PluginInfo[]
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const addGroupButton = useTemplateRef<InstanceType<typeof Button>>('addGroupButton')
const {
  credentialsPath,
  drafts,
  newGroup,
  sections,
  groups,
  declarations,
  deletion,
  actions,
} = useSecretsPage({
  getSecrets: () => props.secrets,
  getPlugins: () => props.plugins,
})

/* -------------------------------- Handlers -------------------------------- */
function onAdd(group: string): void {
  actions.addSecret(group)
}

function onAddGroup(): void {
  actions.addGroup()
}

function onRenameGroup(section: SectionData, name: string): void {
  actions.renameGroup(section, name)
}

function onCreate(
  group: string,
  name: string,
  value: string,
  draftId?: number,
): Promise<boolean> {
  return actions.createSecret(group, name, value, draftId)
}

function onToggle(secret: Secret, declaration: Declaration, isAllowed: boolean): void {
  actions.toggleDeclaration(secret, declaration, isAllowed)
}

function onAskDelete(secret: Secret, opener: HTMLElement): void {
  deletion.ask({ kind: 'secret', secret }, opener)
}

function onAskDeleteGroup(section: SectionData, opener: HTMLElement): void {
  if (section.secrets.length > 0) {
    deletion.ask({ kind: 'group', section }, opener)
    return
  }
  drafts.removeEmptyGroup(section.group)
  runAfterRender(() => (addGroupButton.value?.$el as HTMLElement | undefined)?.focus())
}
</script>

<template>
  <div class="secrets-page">
    <Button variant="ghost" size="small" class="secrets-page__back" @click="emit('back')">
      &larr; {{ words.backToPlugins }}
    </Button>

    <p v-if="credentialsPath !== ''" class="secrets-page__where">
      {{ words.secretsFile(credentialsPath) }}
    </p>

    <SecretSection
      v-for="section in sections"
      :key="section.group"
      :section="section"
      :groups="groups"
      :drafts="drafts.getDrafts(section.group)"
      :declarations="declarations"
      :is-new="section.group === newGroup"
      :create="(name, value, draftId) => onCreate(section.group, name, value, draftId)"
      @add="onAdd(section.group)"
      @rename-group="(name) => onRenameGroup(section, name)"
      @discard="drafts.discardDraft"
      @move="actions.moveSecret"
      @rename="actions.renameSecret"
      @write="actions.writeSecret"
      @delete="onAskDelete"
      @delete-group="(opener) => onAskDeleteGroup(section, opener)"
      @toggle="onToggle"
    />

    <Button
      ref="addGroupButton"
      variant="ghost"
      size="small"
      class="secrets-page__add"
      @click="onAddGroup"
    >
      <Plus :size="14" aria-hidden="true" />
      {{ words.secretAddGroup }}
    </Button>

    <ConfirmDialog
      v-if="deletion.target.value"
      tone="danger"
      :text="deletion.text.value"
      :label="deletion.isGroup.value ? words.secretGroupDelete : words.secretDelete"
      :confirm-text="words.secretDeleteConfirm"
      :cancel-text="words.secretDeleteCancel"
      @confirm="deletion.confirm"
      @cancel="deletion.cancel"
    />
  </div>
</template>

<style scoped>
.secrets-page {
  display: flex;
  flex-direction: column;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.secrets-page__back {
  --margin-end: 0.75rem;
  align-self: flex-start;
  margin-block-end: var(--margin-end);
}

.secrets-page__where {
  margin: 0 0 var(--settings-apart);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  overflow-wrap: anywhere;
}

.secrets-page__add {
  align-self: flex-start;
}
</style>
