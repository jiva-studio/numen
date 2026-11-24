import { inject, provide, ref, type InjectionKey, type Ref } from 'vue'

type Host = Readonly<Ref<HTMLElement | null>>

const DIALOG_HOST: InjectionKey<Host> = Symbol('dialog-host')

/** Makes the element the area that dialogs below cover and are centred in: a tab, a panel. */
export function provideDialogHost(host: Host): void {
  provide(DIALOG_HOST, host)
}

/** The element dialogs are drawn in, null where the window itself is the area. */
export function useDialogHost(): Host {
  return inject(DIALOG_HOST, ref(null))
}
