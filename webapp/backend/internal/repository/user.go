package repository

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/model"
)

type UserRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// mysql> EXPLAIN SELECT user_id, password_hash, user_name FROM users WHERE user_name = 1;
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra       |
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// |  1 | SIMPLE      | users | NULL       | ALL  | NULL          | NULL | NULL    | NULL |  100 |    10.00 | Using where |
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// 1 row in set, 1 warning (0.003 sec)

// ユーザー名からユーザー情報を取得
// ログイン時に使用
func (r *UserRepository) FindByUserName(ctx context.Context, userName string) (*model.User, error) {
	var user model.User
	query := "SELECT user_id, password_hash, user_name FROM users WHERE user_name = ?"

	err := r.db.GetContext(ctx, &user, query, userName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &user, nil
}
