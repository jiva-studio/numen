SELECT path, plugin_id, key, source_version_id, kind, file, input_hash, text_hash, create_time
FROM text_versions
WHERE vault = ? AND plugin_id = ?
ORDER BY id;
