# Local Reranking

Reranks search results with a cross-encoder model on this machine, so that a plugin that reranks texts needs no service and no key.

<!-- more -->

## What You Can Do

- **Rerank without setup**: The plugin adds a connection of the type **Local reranker** to the Connections page as soon as it is enabled. A plugin that reranks texts through its connections can use it at once.
- **Choose a model for each row**: The row of a connection names a model. Two are listed: BGE reranker v2 M3 (the default, 571 MB) and GTE multilingual reranker (smaller and faster, 341 MB, ranks less well). A row with no model runs the default. Any other model needs a profile of your own, described below.
- **Run several calls at once**: The field **Parallel requests** of the connection, from 1 to 4 with the default 1, is how many reranking calls the connection runs at once; the others wait in the queue of the connection, a person's search before background work.

## How it works

Each document is scored against the query as one pair: the tokenizer of the model encodes the pair, cut to the longest pair the profile allows, and the model answers one logit for it. The score is the sigmoid of that logit, a number between 0 and 1, and a higher score means a more relevant document. The scores are returned in the order of the documents. The model runs on the CPU through the ONNX Runtime the host provides, unless the host names another execution provider.

The model is fetched on the first reranking call, at the revision its profile pins, into the models folder every plugin shares, and each file is checked against the sha256 its profile pins; that first call waits for the download. `NUMEN_OFFLINE=1` forbids the download, and a call for a model that is not on the machine then fails so that the next connection of the row list is tried. Listing the models and listing the connection load no model. The model stays loaded until the plugin process stops after ten idle minutes. A call for another model closes the loaded one first.

Neither shipped model was trained with an instruction, so the instruction of a reranking request is ignored.

What fails is written to the log of the plugin with the id of the model: a model that does not download, a file that misses its pinned sum, a model that does not open, a tokenizer that has no template for a pair, and a pass that fails. The call itself fails so that the next connection is tried. A model that no profile lists fails the call without trying the next connection.

## Model profiles

A model is run as its profile says. The profiles that ship with the plugin are in `backend/domain/profile/profiles.json`. To add a model or change one, put a `*.json` file in the `profiles` folder of the plugin data directory (the folder the host grants the plugin as `NUMEN_PLUGIN_DATA_DIR`). A file holds one profile or an array of them, and a profile is merged over the profile with the same `id` or alias, field by field. The files are read at each call. A file that is not valid JSON, names a field the plugin does not know, or holds a profile that cannot run is written to the log of the plugin and skipped whole.

A profile has these fields. `id` is the Hugging Face repository of the model, and the value of the row field Model; `aliases` are other names the row may use, compared ignoring case. `title`, `tier` and `sizeMB` describe it in the list, and a profile with no `tier` is not listed but runs when a row names it. `maxTokens` is the longest pair, special tokens included, that the model reads; it must be above 8. `onnxFile` is the model file in the repository, such as `onnx/model_quantized.onnx`; a file whose name starts with it, such as `model.onnx_data`, is fetched with it. `revision` and `sha256` pin the repository to a commit and each file to its sha256, with the path in the repository as the key, and are optional; a profile that changes `onnxFile` and gives no `sha256` drops the shipped pins.

The model must take `input_ids` and `attention_mask`, and may take `token_type_ids`; a model that takes any other input is refused. It must answer one logit per pair, as `logits` or as its only output; a model that answers more than one logit per pair, such as one with two labels, is refused. The `tokenizer.json` of the repository must have a template for a pair.
