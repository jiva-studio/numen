# Screenshots

Takes the screenshots the stories declare and checks that each has its files. The manual (`apps/docs`), the landing page (`apps/landing`) and the README of each plugin show these pictures.

```
node tools/shoot/shoot.mjs                         # every declared shot
node tools/shoot/shoot.mjs --app docs              # the shots written into apps/docs/src/assets
node tools/shoot/shoot.mjs --plugin views-tasks    # the shots written into plugins/views-tasks/screenshots
node tools/shoot/shoot.mjs --story tasks-status-chip--active
node tools/shoot/check.mjs                         # every declared shot has its files, every plugin picture its story
```

`--app`, `--plugin` and `--story` can be repeated and combined. `npm run shoot` in `apps/docs` and `apps/landing` runs the tool with `--app` of that app.

## Declaring a screenshot

A story is a screenshot when it carries the tag `screenshot` and lists its shots in `parameters.screenshot`:

```ts
export const Active: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [
      { name: 'status', width: 360, height: 48, drawn: 500 },
      { name: 'tasks', app: 'docs' },
    ],
  },
}
```

Each shot writes `<name>-light.webp` and `<name>-dark.webp` into one folder:

- a story under `plugins/<folder>/` writes into `plugins/<folder>/screenshots/`;
- `plugin: '<folder>'` writes into `plugins/<folder>/screenshots/`, for a plugin shown by a story outside it, such as `engines-ocr` by the window story `application-window--recognising`;
- `app: 'docs'` writes into `apps/docs/src/assets/`, `app: 'landing'` into `apps/landing/src/assets/`.

A story outside `plugins/` names a plugin or an app in every shot.

| Key | Meaning |
| --- | --- |
| `name` | The file name before `-light.webp` and `-dark.webp`: lower-case words joined by dashes. |
| `plugin`, `app` | The folder the files are written into, as above. |
| `width`, `height` | The window the story is drawn in, in CSS pixels. 1180 by 740 when not given. |
| `over` | A selector the pointer rests on before the picture is taken. |
| `preset` | A palette of `plugins/themes-presets/themes/` the window is drawn with. |
| `drawn` | The fewest bytes the picture comes to; a smaller one fails as a story that did not draw. 15000 when not given. |

The values are literals: the check reads the story sources with the TypeScript parser and does not run them.

## A plugin inside the window

A plugin's picture shows the plugin in the window. A story in the plugins' Storybook that sets `parameters.frame` is drawn inside a stand-in of the window: the title bar, the file tree of an invented vault of physics notes, the panes with their tabs, and the status bar. The frame is `libs/plugin/sdk-ts/stories/.storybook/frame/`; a plugin imports nothing of it, since a plugin reaches only `@numen/plugin-sdk` and `@numen/ui`.

```ts
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'tab', tab: { title: 'Boltzmann 1877', icon: 'book' }, note: 'after' },
    screenshot: [{ name: 'reader' }],
  },
}
```

| `place` | Where the story is drawn |
| --- | --- |
| `tab` | In the pane of the tab `tab`, in front of the tabs `otherTabs`, with a note in a pane `before` or `after` it taking `1 - share` of the width. |
| `files` | In place of the file tree. |
| `title-bar`, `status-bar` | In place of the bar. |
| `title-status` | In the status slot at the end of the title bar; a popover it opens is drawn over the window. |

`notePath` names the note of `vault.ts` the note pane shows, Entropy when not given. `selected` chooses a row of the file tree by its path, the shown note when not given, `files: false` draws the window without it, and `title` names the vault in the title bar. The window shots of `@numen/workbench` draw the same bars with `hasBars` of `Window.stories.ts`.

## Taking them

Each shot is drawn in the Storybook that declares its story: the components' in `libs/ui/components`, the shell's in `libs/ui/workbench`, the editor's in `apps/desktop/editor`, or the plugins' in `libs/plugin/sdk-ts/stories`. The run starts the Storybooks its shots need and stops them afterwards. `STORYBOOK_PORT`, `STORYBOOK_SHELL_PORT`, `STORYBOOK_EDITOR_PORT` and `STORYBOOK_PLUGINS_PORT` name Storybooks already running. The shell's and the plugins' Storybooks use `@numen/ui` and `@numen/workbench` through their builds: run `npm run build` in `libs/ui/components`, then in `libs/ui/workbench`, first.

Every shot is taken in light and in dark at 2.5 device pixels per CSS pixel and written as WebP. A picture whose pixels are unchanged is not rewritten, because the encoder writes different bytes for the same image. A story that fails to draw is reported at the end with the others; the rest are still taken.

The browser is `CHROME_PATH` when set, else a Chrome on `PATH`, else Playwright's own.

Change what is in a picture by changing the story, not the file.

## The check

`check.mjs` fails when a declaration is wrong (an unknown key, a missing plugin or palette, a story outside `plugins/` without a target, two shots writing one file, a tag without shots or shots without the tag), when a declared shot has no files, and when a file under `plugins/*/screenshots/` is written by no declared shot. CI runs it in the `references` checks.
