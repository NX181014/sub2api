UPDATE proxies AS p
SET status = 'disabled',
    deleted_at = COALESCE(p.deleted_at, NOW()),
    updated_at = NOW()
FROM mihomo_routes AS r
WHERE r.proxy_id = p.id
  AND r.deleted_at IS NOT NULL
  AND p.deleted_at IS NULL
  AND p.managed_source LIKE 'mihomo:%'
  AND NOT EXISTS (
      SELECT 1
      FROM accounts AS a
      WHERE a.proxy_id = p.id
        AND a.deleted_at IS NULL
  );
