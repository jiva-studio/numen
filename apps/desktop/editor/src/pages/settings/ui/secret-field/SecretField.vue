<script setup lang="ts">
/**
 * The secret one declaration of a plugin reads: a select of the secrets, each under the name of
 * its group. Choosing a secret allows the declaration to read it, or stops it, as the check box
 * on the page of secrets does. Where the declared group has no secret, an icon button beside the
 * name opens the page of secrets.
 */
import { computed } from 'vue'
import { KeyRound } from '@lucide/vue'
import { getServiceInfo } from '@numen/core-client'
import { Button, Select, type SelectChoice } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import {
  getMenuSecrets,
  hasGroupSecrets,
  isAllowed,
  type Declaration,
  type SecretSection,
} from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  declaration: Declaration
  sections: readonly SecretSection[]
}>()

const emit = defineEmits<{
  (e: 'toggle', secret: Secret, isAllowed: boolean): void
  (e: 'open-secrets'): void
}>()

/* --------------------------------- State ---------------------------------- */
const secrets = computed(() => getMenuSecrets(props.sections, props.declaration))
const choices = computed<readonly SelectChoice[]>(() =>
  secrets.value.map((one) => ({
    id: one.id,
    text: one.name,
    ...(one.group === '' ? {} : { detail: getServiceInfo(one.group).title || one.group }),
  })),
)
const allowedIds = computed(() =>
  secrets.value.filter((one) => isAllowed(one, props.declaration)).map((one) => one.id),
)
/** The id SettingRow gives the name it labels the field by. */
const labelledBy = computed(() => `settings-${props.at}`)
const hasOwnSecrets = computed(() => hasGroupSecrets(props.sections, props.declaration))

/* -------------------------------- Handlers -------------------------------- */
/** The select hands on the new list of allowed ids: the one that came or went is the one toggled. */
function onSelect(next: readonly string[]): void {
  const added = next.find((id) => !allowedIds.value.includes(id))
  const id = added ?? allowedIds.value.find((one) => !next.includes(one))
  const secret = secrets.value.find((one) => one.id === id)
  if (secret) emit('toggle', secret, added !== undefined)
}
</script>

<template>
  <SettingRow :at="at" :name="name" :detail="detail">
    <template v-if="!hasOwnSecrets && declaration.group !== ''" #action>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretOpenPage"
        :title="words.secretOpenPage"
        @click="emit('open-secrets')"
      >
        <KeyRound :size="14" aria-hidden="true" />
      </Button>
    </template>

    <Select
      class="settings__secret"
      is-multiple
      :selected="allowedIds"
      :choices="choices"
      :placeholder="words.secretFieldNone"
      :name="words.secretChoose"
      :search-name="words.secretSearch"
      :aria-labelledby="labelledBy"
      @update:selected="onSelect"
    />
  </SettingRow>
</template>

<style scoped>
.settings__secret {
  inline-size: var(--settings-choice);
  max-inline-size: 100%;
}
</style>
