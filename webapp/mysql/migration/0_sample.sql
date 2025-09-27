-- このファイルに記述されたSQLコマンドが、マイグレーション時に実行されます。
CREATE INDEX idx_users_covering ON users(user_name, user_id, password_hash);