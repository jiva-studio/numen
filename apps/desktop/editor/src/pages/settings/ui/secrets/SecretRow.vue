<script setup lang="ts">
/**
 * One secret of a group as a row: its name as an input where the group holds more than one secret, the value
 * (dots as the placeholder while one is stored), the plugins that may read it, and the menu to move and delete it.
 */
import { ref, watch } from 'vue'
import type { Secret } from '@/entities/settings'
import type { Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SecretNameInput from './SecretNameInput.vue'
import SecretPermissions from './SecretPermissions.vue'
import SecretRowMenu from './SecretRowMenu.vue'
import SecretRowFrame from './SecretRowFrame.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret: Secret
  isFirst: boolean
  isLast: boolean
  /** Whether the group holds more than one secret, so that the order can change. */
  canMove: boolean
  /** Whether the row shows its name, as the group holds more than one secret. */
  hasName: boolean
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
const draftName = ref(props.secret.name)

watch(
  () => props.secret.name,
  (name) => {
    draftName.value = name
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onCommitRename(): void {
  const name = draftName.value.trim()
  if (name !== '' && name !== props.secret.name) emit('rename', name)
  else draftName.value = props.secret.name
}
</script>

<template>
  <SecretRowFrame :has-name="hasName">
    <template #name>
      <SecretNameInput
        v-model="draftName"
        @commit="onCommitRename"
        @cancel="draftName = secret.name"
      />
    </template>

    <template #value>
      <SecretValueInput
        :label="words.secretValue(secret.name)"
        :placeholder="secret.hasValue ? words.secretMasked : words.keyPlaceholder"
        @commit="(value) => emit('write', value)"
      />
    </template>

    <template #permissions>
      <SecretPermissions
        :secret="secret"
        :declarations="declarations"
        @toggle="(declaration, isAllowed) => emit('toggle', declaration, isAllowed)"
      />
    </template>

    <template #actions>
      <SecretRowMenu
        :is-first="isFirst"
        :is-last="isLast"
        :can-move="canMove"
        @move-up="emit('move-up')"
        @move-down="emit('move-down')"
        @delete="(opener) => emit('delete', opener)"
      />
    </template>
  </SecretRowFrame>
</template>
