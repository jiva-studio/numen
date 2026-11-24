import { inject, provide, type InjectionKey, type Ref } from 'vue'

const DIALOG_HOST: InjectionKey<Readonly<Ref<HTMLElement | null>>> = Symbol('dialog-host')

/** Makes the element the area that dialogs below cover and are centred in: a tab, a panel. */
export function provideDialogHost(host: Readonly<Ref<HTMLElement | null>>): void {
  provide(DIALOG_HOST, host)
}

/** The element dialogs are drawn in, or null where the window itself is the area. */
export function useDialogHost(): HTMLElement | null {
  return inject(DIALOG_HOST, null)?.value ?? null
}
