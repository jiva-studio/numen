<script setup lang="ts">
/**
 * The recovery kit of an identity: the button to print a kit, the proof asked for before it, the
 * three sheets with their buttons, and the check of two sheets against the account.
 */
import { Button } from '@numen/ui'
import type { useRecoveryKit } from '../../model/useRecoveryKit'
import { WORDS as words } from '../../words'
import KitCheck from './KitCheck.vue'
import KitProof from './KitProof.vue'
import KitSheets from './KitSheets.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  model: ReturnType<typeof useRecoveryKit>
  texts: readonly [string, string]
  isBusy: boolean
}>()
</script>

<template>
  <div class="recovery-kit">
    <KitProof
      v-if="model.step.value === 'proof'"
      :words-text="model.wordsText.value"
      :texts="texts"
      :is-busy="isBusy"
      @change-words="model.edit.setWordsText"
      @change-sheet="model.edit.setSheetText"
      @make="model.actions.makeKit"
      @cancel="model.actions.close"
    />
    <div v-if="model.kit.value" class="recovery-kit__shown">
      <h3 class="recovery-kit__title">{{ words.kitTitle }}</h3>
      <p class="recovery-kit__note">{{ words.kitIntro }} {{ words.kitGone }}</p>
      <KitSheets :kit="model.kit.value" />
    </div>
    <KitCheck
      v-if="model.checking.isChecking.value"
      :texts="texts"
      :is-fitting="model.checking.isFitting.value"
      :is-busy="isBusy"
      @change="model.edit.setSheetText"
      @check="model.checking.check"
      @cancel="model.checking.cancel"
    />
    <div v-if="model.step.value !== 'proof'" class="recovery-kit__actions">
      <Button v-if="model.kit.value" type="button" variant="solid" @click="model.actions.print">
        {{ words.kitPrint }}
      </Button>
      <Button
        v-else
        type="button"
        variant="outline"
        :disabled="isBusy"
        @click="model.actions.openProof"
      >
        {{ words.kitPrintNew }}
      </Button>
      <Button
        v-if="!model.checking.isChecking.value"
        type="button"
        variant="outline"
        :disabled="isBusy"
        @click="model.checking.open"
      >
        {{ words.kitCheck }}
      </Button>
      <Button v-if="model.kit.value" type="button" variant="outline" @click="model.actions.close">
        {{ model.isFirst.value ? words.kitSkip : words.kitDone }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.recovery-kit {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.recovery-kit__title {
  margin: 0;
  font-size: var(--numen-text-2);
  font-weight: 500;
}

.recovery-kit__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.recovery-kit__shown {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.recovery-kit__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
