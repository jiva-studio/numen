---
title: Command-line options
description: The flags numen takes when it starts, and what each is for.
---

numen opens the vault you had last and reads its settings from the usual place. The flags below
are for the times you want a single launch to do something else; none of them is written down,
and the next launch is back to normal.

```sh
numen -vault Research
```

<!-- BEGIN AUTOGEN -->
| | |
| --- | --- |
| `-index` | Path to the index database. |
| `-registry` | Path to the vault list. |
| `-vault` | The vault to open: a name, a path or an identity; the one opened last by default. |
| `-mcp-addr` | Where agents reach this vault; anything but a loopback address opens it to the network. |
| `-no-mcp` | Do not let agents reach this vault. |
| `-interface-scale` | How large the interface is drawn, 1 being as designed; this launch alone. |
| `-text-scale` | How large the text a person reads is set, 1 being as designed; this launch alone. |
| `-rebuild-index` | Read every file and put it in the index again, whatever the index remembers. |
| `-version` | Say what this build is and stop. |
<!-- END AUTOGEN -->

## The ones worth knowing

**`-vault`** takes a name, a path or a vault's identity. It is the quickest way to keep two
vaults open in two windows, or to put a shortcut on the desktop that opens the one you work in.

**`-rebuild-index`** reads every file again and puts it back in the index, whatever the index
thought it already knew. Nothing in your vault is touched — the index is built from your files
and can always be built again. Reach for it when search is answering with something stale.

**`-interface-scale`** and **`-text-scale`** try a size out for one launch. Choosing a size in
the window is what writes it down; these do not.

**`-no-mcp`** starts the window with no door for agents at all. Without it, agents reach this
vault at `127.0.0.1:7717`, which is this machine and nothing else; `-mcp-addr` moves that, and
anything other than a loopback address opens the vault to the network.

The other program, [`numen-cli`](/cli/), takes flags of its own.
