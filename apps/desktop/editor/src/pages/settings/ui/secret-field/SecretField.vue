<script setup lang="ts">
/**
 * The secret one declaration of a plugin reads: a select of the secrets, each under the name of
 * its group. Choosing a secret allows the declaration to read it, or stops it, as the check box
 * on the page of secrets does. Where the declared group has no secret, an icon button beside the
 * name opens the page of secrets.
 */
import { computed } from 'vue'
import { KeyRound } from '@lucide/vue'
import { Button } from '@numen/ui'
import { getServiceInfo, isAllowed, type Secret, type SecretMenu } from '@/entities/settings'
import {
  getMenuSecrets,
  hasGroupSecrets,
  type Declaration,
  type SecretSection,
} from '../../lib/secretSections'
import type { SelectChoice } from '../../types'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'
import { MultiSelect } from '../multi-select'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  detail: string
  declaration: Declaration
  sections: readonly SecretSection[]
  menus: readonly SecretMenu[]
}>()

const emit = defineEmits<{
  (e: 'toggle', secret: Secret, isAllowed: boolean): void
  (e: 'open-secrets'): void
}>()

/* --------------------------------- State ---------------------------------- */
const secrets = computed(() => getMenuSecrets(props.menus, props.declaration))
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
function onToggle(id: string, isOn: boolean): void {
  const secret = secrets.value.find((one) => one.id === id)
  if (secret) emit('toggle', secret, isOn)
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

    <MultiSelect
      class="settings__secret"
      :selected="allowedIds"
      :choices="choices"
      :placeholder="words.secretFieldNone"
      :name="words.secretChoose"
      :search-name="words.secretSearch"
      :get-more-text="words.secretMore"
      :aria-labelledby="labelledBy"
      @toggle="onToggle"
    />
  </SettingRow>
</template>
