-- 商品検索の最適化
CREATE INDEX idx_products_name_prefix ON products(name(20));
CREATE INDEX idx_products_description_prefix ON products(description(50));

-- 注文検索の最適化
CREATE INDEX idx_orders_user_product_name ON orders(user_id) 
COMMENT 'orders と products の JOIN 最適化';

-- covering index for order queries
CREATE INDEX idx_orders_covering ON orders(user_id, created_at, order_id, product_id, shipped_status);