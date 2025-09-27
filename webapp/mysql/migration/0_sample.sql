-- このファイルに記述されたSQLコマンドが、マイグレーション時に実行されます。
CREATE INDEX idx_users_covering ON users(user_name, user_id, password_hash);

-- Index for session_uuid lookups (most important)
CREATE INDEX idx_user_sessions_uuid_expires ON user_sessions(session_uuid, expires_at);

-- Create covering index
CREATE INDEX idx_user_sessions_covering ON user_sessions(session_uuid, expires_at, user_id);

-- Index for cleaning up expired sessions
CREATE INDEX idx_user_sessions_expires_cleanup ON user_sessions(expires_at);

-- For search performance
CREATE INDEX idx_products_name_search ON products(name);
CREATE INDEX idx_products_description_search ON products(description);

-- For sorting performance
CREATE INDEX idx_products_value ON products(value, product_id);
CREATE INDEX idx_products_weight ON products(weight, product_id);
CREATE INDEX idx_products_name_sort ON products(name, product_id);

-- Composite index for user orders with common sort fields
CREATE INDEX idx_orders_user_created ON orders(user_id, created_at, order_id);
CREATE INDEX idx_orders_user_status ON orders(user_id, shipped_status, order_id);
CREATE INDEX idx_orders_user_arrived ON orders(user_id, arrived_at, order_id);

-- Index for product name searches
CREATE INDEX idx_products_name_search ON products(name);