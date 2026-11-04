---
title: Command-line options
description: The flags numen takes when it starts, and what each is for.
---

numen opens the vault you had last and reads its settings from the usual place. The flags below
are for the times you want a single launch to do something else; none of them is saved,
and the next launch uses the normal settings.

```sh
numen -vault Research
```

<!-- BEGIN AUTOGEN -->
| | |
| --- | --- |
| `-index` | Path to the index database. |
| `-registry` | Path to the vault list. |
| `-vault` | The vault to open: a name, a path or an identity; the one opened last by default. |
| `-interface-scale` | Scale of the interface (controls and spacing); 1 is the default; this launch only. |
| `-text-scale` | Scale of note and book text; 1 is the default; this launch only. |
| `-rebuild-index` | Read every file again and rewrite its index entries, whatever the index already holds. |
| `-safe-mode` | Disable plugins for emergency recovery. |
| `-version` | Print the version and exit. |
<!-- END AUTOGEN -->

## The ones worth knowing

**`-vault`** takes a name, a path or a vault's identity. It is the quickest way to keep two
vaults open in two windows, or to put a shortcut on the desktop that opens the one you work in.

**`-rebuild-index`** reads every file again and puts it back in the index, regardless of what the index
already contained. Nothing in your vault is touched: the index is built from your files
and can always be built again. Use it when search returns stale results.

**`-interface-scale`** and **`-text-scale`** try a size out for one launch. Choosing a size in
the window is what saves it; these do not.

The other program, [`numen-cli`](/cli/), takes flags of its own.
