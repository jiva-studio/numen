// The browser commands the shared story setup registers (libs/ui/components/.storybook/browser.ts),
// declared for plugin stories that call them.
export {}

declare module 'vitest/browser' {
  interface BrowserCommands {
    setSystemScheme: (scheme: 'light' | 'dark' | null) => Promise<void>
  }
}
