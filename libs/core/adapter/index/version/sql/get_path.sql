SELECT path, plugin_id, key, source_version_id, kind, file, source_hash, hash, create_time
FROM text_versions
WHERE vault = ? AND path = ?
ORDER BY id;
