-- orders テーブルの user_id と created_at にインデックスを追加
CREATE INDEX idx_orders_user_id_created_at ON orders(user_id, created_at);

-- orders テーブルの shipped_status にインデックスを追加
CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);

-- products テーブルの name と description に FULLTEXT インデックスを追加
-- 注意: FULLTEXT インデックスは部分一致検索 (`LIKE '%word%'`) には効果がありませんが、
--       `MATCH(...) AGAINST(...)` 構文を使うことで高速な全文検索が可能です。
--       今回は `LIKE` が使われているため、通常のインデックスも検討の価値があります。
CREATE INDEX idx_products_name_description ON products(name, description);

-- users テーブルの user_name にユニークインデックスを追加
CREATE UNIQUE INDEX idx_users_user_name ON users(user_name);

-- user_sessions テーブルの session_uuid にインデックスを追加
CREATE INDEX idx_user_sessions_session_uuid ON user_sessions(session_uuid);