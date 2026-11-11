import { computed, type ComputedRef } from 'vue'

/** The name of the card model a card is cut by, or what the words say for a card cut by none. */
export function useCardModelTitle(
  getCardModel: () => string | null,
  getSayUnknown: () => (cardModel: string | null) => string,
): ComputedRef<string> {
  return computed(() => getCardModel() ?? getSayUnknown()(null))
}
