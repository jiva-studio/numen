/** The entries of the question about how to keep the keys: the way, the passphrase and its repeat. */
import { computed, ref } from 'vue'
import { isPassphraseMismatch, isPassphraseSet } from '../lib/isPassphraseSet'

export function useKeyFileChoice() {
  const mode = ref<'passphrase' | 'plaintext'>('passphrase')
  const passphrase = ref('')
  const confirmation = ref('')
  const isComplete = computed(
    () => mode.value === 'plaintext' || isPassphraseSet(passphrase.value, confirmation.value),
  )
  const isMismatch = computed(
    () => mode.value === 'passphrase' && isPassphraseMismatch(passphrase.value, confirmation.value),
  )
  return { mode, passphrase, confirmation, isComplete, isMismatch }
}
