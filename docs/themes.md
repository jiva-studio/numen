# Themes

A theme is one CSS file that redeclares the tokens the interface draws with. The default theme ships inside the application, plugins contribute more, and the rest are files in a folder of the person's; they are one list, one format, and one line applies any of them.

Every token a theme leaves out keeps the value set in `tokens.css`. A file that sets one token is a theme.

## The file

```css
:root {
  --numen-surface: #101014;
  --numen-raised: #17171d;
  --numen-ink: #e6e6ea;
}
```

Every token is on the root, and a component inherits what it paints with, so a `:root` block applies to every component. A custom property declared on an element overrides the same property inherited from the root, regardless of selector specificity, so `:root` is where a theme is written.

It is an ordinary stylesheet, applied whole. The application reads only whether it sets `color-scheme`. If a person's own file hides the window, `app.appearance.theme` in the settings selects another theme.

## Light and dark

A palette published in two halves writes `light-dark()` pairs and does not set `color-scheme`:

```css
:root {
  --numen-surface: light-dark(#fdf6e3, #002b36);
}
```

`app.appearance.mode` — `system`, `light`, `dark` — selects which half is used.

A palette published in one half writes flat colours and pins the half it is:

```css
:root {
  color-scheme: dark;
  --numen-surface: #21222c;
}
```

Such a theme is listed as pinned, and the mode has no effect. For a theme in the person's folder, what counts as pinning is the property with a colon after it, anywhere in the file; the comments are removed first, so a comment saying `color-scheme` pins nothing. A theme a plugin contributes is pinned when its manifest declares an `appearance` of `light` or `dark`; its file is not read to find out.

## The window opens with the theme applied

`index.html` is not served by the file server. A handler reads it and splices three elements into the end of the head — which half of a colour pair the tokens are read as, the theme's own file, and the two sizes — then writes the page. No frame is drawn in the default colours.

Each has `data-appearance`, which is `mode`, `theme` or `sizes`, and the window uses it to find each of them. They come after everything the build put in the head, the built stylesheet's link last among them, and they are read for every request: a theme chosen, a size chosen, or a file in the person's folder saved over is applied by the next reload. A build with no interface, and a page with no head, are served as the file server has them.

## Where themes are stored

The themes folder is `numen/themes/` in the folder this desktop keeps a person's configuration in, beside `numen.json` — see [Settings](settings.md). It is created, empty, on the first run.

|              |                                                                                                                     |
| ------------ | ------------------------------------------------------------------------------------------------------------------- |
| One level    | a folder inside the themes folder is not a theme, and neither is a file one level down.                             |
| `.css`       | `dracula.css.bak` and `README.md` are files in the folder and not themes.                                           |
| 262144 bytes | a file larger than that is neither offered nor read. Every shipped palette is under two kilobytes.                  |
| Readable     | a broken link, a file the disk refuses to open: not in the list. A name in the list is a theme that can be applied. |

A folder that is not there or cannot be read leaves only the default theme and the themes of the plugins in the list.

The folder is watched at that one level: a theme written, saved over or removed there is detected by name immediately.

## What a theme is called

`preset:numen` ships inside the application; `mine:dracula` is the file `dracula.css` in the themes folder; `plugin:md.numen.themes.presets/dracula` is the theme `dracula` of the plugin `md.numen.themes.presets`. Two origins may carry one title, and neither replaces the other.

The half after `mine:` is one filename in that folder. `mine:../../../.ssh/id_rsa`, `mine:..`, `dracula` and `originless:dracula` are not themes. The half after `plugin:` is a plugin id and a theme id; the theme is looked up in the manifests of the enabled plugins and no path is built from the name.

A name matching nothing falls back to `preset:numen`, and the name that was not found is reported. The settings are left as they are: putting the file back, or enabling the plugin again, is enough. A name from an earlier release, such as `preset:dracula`, matches nothing; choose the theme again.

## Themes from a plugin

A plugin lists its themes in `contributes.themes` of its manifest:

```json
{
  "id": "org.example.themes",
  "name": "Example Themes",
  "targets": ["desktop.editor", "desktop.flashcards"],
  "contributes": {
    "themes": [{ "id": "night", "name": "Night", "appearance": "dark", "file": "themes/night.css" }]
  }
}
```

|              |                                                                                                                                   |
| ------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `id`         | a lowercase slug, unique within the plugin.                                                                                       |
| `name`       | the title the theme is listed under.                                                                                              |
| `appearance` | `light` or `dark` for a palette published in one half, and left out for a palette in two halves.                                  |
| `file`       | a `.css` file inside the plugin folder: a relative path, no `..`, at most 262144 bytes. A file that is not there fails discovery. |

A plugin of themes has no backend and no frontend; it consists only of the manifest and the stylesheets. The list is read from the manifests discovered at start, so the themes are available before any plugin process starts, and the page is served with the theme applied from the first frame. Only the enabled plugins whose `targets` include the window offer their themes: disabling the plugin removes its themes from the list, a window that applies one falls back to `preset:numen`, and a window open at the time is notified which themes were added or removed. A file is read through the plugin folder only; a link that leaves the folder is not followed. The editor lists a plugin's themes under the plugin's name. The mobile app has no plugins and offers the default theme and the person's own.

## The tokens a theme sets

Forty-four, each a colour unless stated otherwise.

### The window and what is on it

|                      |                                                                                                      |
| -------------------- | ---------------------------------------------------------------------------------------------------- |
| `--numen-surface`    | the background colour of the window.                                                                 |
| `--numen-raised`     | the fill of a node, a card and a pane.                                                               |
| `--numen-ink`        | text on that fill, and the caret in it.                                                              |
| `--numen-rule`       | the border around it.                                                                                |
| `--numen-accent`     | the node that has keyboard focus, the fill of a button, and every line and mark drawn in the accent. |
| `--numen-accent-ink` | text on the accent.                                                                                  |
| `--numen-ring`       | the outline around what is focused, and a link in marked-up text.                                    |
| `--numen-alarm`      | the error colour: an error, and text the parser could not read.                                      |
| `--numen-alarm-bg`   | the background of a card that shows an alarm.                                                        |
| `--numen-hushed`     | muted text next to the content it describes: a caption, a count, a month over a grid.                |

### The plex

|                              |                                                                                        |
| ---------------------------- | -------------------------------------------------------------------------------------- |
| `--numen-relation-parent`    | the hue of a parent's relation.                                                        |
| `--numen-relation-child`     | the hue of a child's relation.                                                         |
| `--numen-relation-jump`      | the hue of a jump's relation.                                                          |
| `--numen-relation-reference` | the hue of a reference's relation.                                                     |
| `--numen-relation-sibling`   | the hue of a sibling's relation.                                                       |
| `--numen-edge`               | the line drawn between two nodes.                                                      |
| `--numen-edge-label`         | the text on that line, which uses the hushed colour unless a theme sets it separately. |

### Text being read, and text about to change

|                           |                                                                                                                        |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `--numen-selection`       | the background of selected text.                                                                                       |
| `--numen-selection-away`  | the same selection where the keyboard focus is elsewhere.                                                              |
| `--numen-selection-match` | every other occurrence of that text.                                                                                   |
| `--numen-highlight`       | the background of a range that is about to change. It is drawn under words that are still being read, so it is a tint. |
| `--numen-caution-bg`      | the background of a notice that a person should read but need not act on at once.                                      |
| `--numen-caution-fg`      | the text in that notice.                                                                                               |
| `--numen-positive`        | the colour of work that is going well and is still in progress.                                                        |

### Marked-up text

|                              |                                                      |
| ---------------------------- | ---------------------------------------------------- |
| `--numen-syntax-mark`        | the marks themselves, shown under the caret.         |
| `--numen-code-bg`            | the background of code.                              |
| `--numen-table-head-bg`      | the background of the head row of a table.           |
| `--numen-syntax-keyword`     | keywords, modifiers, and operators written as words. |
| `--numen-syntax-name`        | names: variables, properties, attributes.            |
| `--numen-syntax-type`        | type names, class names, namespaces, tags.           |
| `--numen-syntax-string`      | strings and regular expressions.                     |
| `--numen-syntax-number`      | numbers, booleans, atoms, escapes.                   |
| `--numen-syntax-comment`     | comments, which are set in italic.                   |
| `--numen-syntax-punctuation` | operators, brackets, separators.                     |

### What floats over the window

|                        |                                                                                                                                           |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `--numen-panel-bg`     | the tint of a panel.                                                                                                                      |
| `--numen-panel-border` | the line around it.                                                                                                                       |
| `--numen-panel-blur`   | a length: how much what shows through the tint is blurred. An opaque tint leaves nothing to blur, so a theme setting one sets the other.  |
| `--numen-panel-shadow` | the shadow of a panel. It contains its offsets and its spread besides its colour, so a theme that sets it must write those lengths again. |
| `--numen-shadow-card`  | the shadow of a card, written the same way.                                                                                               |
| `--numen-scrim`        | the overlay colour a panel draws over the content it covers.                                                                              |

### Fields, and what is said in them

|                        |                                                     |
| ---------------------- | --------------------------------------------------- |
| `--numen-field-bg`     | the background of a text field.                     |
| `--numen-field-border` | the line around it.                                 |
| `--numen-bubble-bg`    | the background of a chat bubble.                    |
| `--numen-bubble-fg`    | the text in it.                                     |
| `--numen-answer-fg`    | the text colour of an answer, drawn on the surface. |

## What a theme leaves alone

The type, the measures and the timings: the font stack, the two text sizes and the line height, the radii and strokes, the spacing, the sizes of a handle and a field, the stacking order, the durations and the easing. They are the same under every palette, and `tokens.css` holds them under the palette in three groups — what follows the interface, what the text multiplier also applies to, and what follows neither.

Nothing restricts a theme to the palette. A theme may declare `--numen-font-size`, and a length written in `rem` still follows the interface.

A theme does not change the two multipliers. `--numen-interface-scale` and `--numen-text-scale` are declared in the last element of the head, after the mode's and the theme's, so a theme that sets either has the earlier of two declarations of equal weight and the window is drawn at the size a person chose.

## What ships

The application ships `preset:numen`, this product's own palette in two halves. The plugin `md.numen.themes.presets` (Preset Themes) contains the others, for the editor and the flashcards window:

|               | Name in the plugin                                          |
| ------------- | ----------------------------------------------------------- |
| `catppuccin`  | Latte in the light, Mocha in the dark.                      |
| `github`      | Light Default in the light, Dark Default in the dark.       |
| `gruvbox`     | two halves, orange accent.                                  |
| `solarized`   | two halves, one set of accents across both.                 |
| `ayu`         | two halves, yellow accent across both.                      |
| `tokyo-night` | Tokyo Night in the dark, Tokyo Night Light in the light.    |
| `dracula`     | dark, pinned.                                               |
| `nord`        | dark, pinned.                                               |
| `one-dark`    | dark, pinned: Atom's hues on One Dark Pro's grounds.        |
| `monokai`     | dark, pinned: the original, as Visual Studio Code sets it.  |
| `cobalt2`     | dark, pinned: yellow on navy.                               |
| `amber`       | dark, pinned: four colours, and the rest derived from them. |

Each is its publisher's palette from that publisher's own source; what a palette does not name is derived from what it does, and the file says which. Each sets the colours its publisher defines and leaves the rest — marked-up text among them — as `tokens.css` has it.

The files are in `plugins/themes-presets/themes/`. A first theme is one of them copied into the themes folder under its own name: `mine:dracula` is in the list beside `plugin:md.numen.themes.presets/dracula`.

## Choosing one

The editor offers the theme and the mode in the settings and in the command palette (Theme, Light or dark). The flashcards window has the same two choices in its command palette, opened with Ctrl+P (Cmd+P): the themes grouped by origin, with a plugin's themes under the plugin's name, and the three modes, which are not offered while the applied theme is pinned.

## What is stored

```json
{
  "app": {
    "appearance": {
      "mode": "system",
      "theme": "plugin:md.numen.themes.presets/dracula"
    }
  }
}
```

Both fields are in `numen.json`, described in [Settings](settings.md).
