import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

/** A flag that is true at once where the delay is none, else turns true once the delay has passed after mount. */
export function useDelayedFlag(getDelay: () => number): Readonly<Ref<boolean>> {
  const isOn = ref(getDelay() <= 0)
  let timer: ReturnType<typeof setTimeout> | undefined
  onMounted(() => {
    if (!isOn.value) timer = setTimeout(() => (isOn.value = true), getDelay())
  })
  onBeforeUnmount(() => clearTimeout(timer))
  return isOn
}
