-- -- webapp/mysql/migration/1_create_indexes.sql

-- -- 商品テーブルに全文検索インデックスを追加
-- ALTER TABLE products ADD FULLTEXT INDEX idx_products_name_description (name, description) WITH PARSER ngram;

-- -- ユーザーテーブルのuser_nameにインデックスを追加（ログイン高速化のため）
-- CREATE INDEX idx_users_user_name ON users(user_name);

-- -- ordersテーブルのshipped_statusにインデックスを追加（ロボットAPI高速化のため）
-- CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);