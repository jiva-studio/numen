import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'md.numen.spike',
  appName: 'numen',
  webDir: 'dist',
  android: { allowMixedContent: true },
  server: { androidScheme: 'http', cleartext: true },
}

export default config
