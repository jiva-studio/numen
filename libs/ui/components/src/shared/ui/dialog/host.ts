import { inject, provide, ref, type InjectionKey, type Ref } from 'vue'

type Host = Readonly<Ref<HTMLElement | null>>

const DIALOG_HOST: InjectionKey<Host> = Symbol('dialog-host')

/** Makes the element the area that every dialog below covers and is centred in: the application content between the title bar and the status bar. */
export function provideDialogHost(host: Host): void {
  provide(DIALOG_HOST, host)
}

/** The element dialogs are drawn in, null where the window itself is the area. */
export function useDialogHost(): Host {
  return inject(DIALOG_HOST, ref(null))
}
