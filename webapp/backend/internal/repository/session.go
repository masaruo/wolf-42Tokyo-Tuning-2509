package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SessionRepository struct {
	db DBTX
}

func NewSessionRepository(db DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}


// mysql> EXPLAIN INSERT INTO user_sessions (session_uuid, user_id, expires_at) VALUES (1, 1, 1);
// +----+-------------+---------------+------------+------+---------------+------+---------+------+------+----------+-------+
// | id | select_type | table         | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra |
// +----+-------------+---------------+------------+------+---------------+------+---------+------+------+----------+-------+
// |  1 | INSERT      | user_sessions | NULL       | ALL  | NULL          | NULL | NULL    | NULL | NULL |     NULL | NULL  |
// +----+-------------+---------------+------------+------+---------------+------+---------+------+------+----------+-------+
// 1 row in set, 1 warning (0.003 sec)
// セッションを作成し、セッションIDと有効期限を返す

func (r *SessionRepository) Create(ctx context.Context, userBusinessID int, duration time.Duration) (string, time.Time, error) {
	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(duration)
	sessionIDStr := sessionUUID.String()

	query := "INSERT INTO user_sessions (session_uuid, user_id, expires_at) VALUES (?, ?, ?)"
	_, err = r.db.ExecContext(ctx, query, sessionIDStr, userBusinessID, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return sessionIDStr, expiresAt, nil
}

// +----+-------------+-------+------------+--------+----------------------+---------+---------+--------------------------+------+----------+-------------+
// | id | select_type | table | partitions | type   | possible_keys        | key     | key_len | ref                      | rows | filtered | Extra       |
// +----+-------------+-------+------------+--------+----------------------+---------+---------+--------------------------+------+----------+-------------+
// |  1 | SIMPLE      | s     | NULL       | ALL    | session_uuid,user_id | NULL    | NULL    | NULL                     |  367 |     3.33 | Using where |
// |  1 | SIMPLE      | u     | NULL       | eq_ref | PRIMARY              | PRIMARY | 4       | 42tokyo2508-db.s.user_id |    1 |   100.00 | Using index |
// +----+-------------+-------+------------+--------+----------------------+---------+---------+--------------------------+------+----------+-------------+
// 2 rows in set, 4 warnings (0.002 sec)
// セッションIDからユーザーIDを取得

// func (r *SessionRepository) FindUserBySessionID(ctx context.Context, sessionID string) (int, error) {
// 	var userID int
// 	query := `
// 		SELECT 
// 			u.user_id
// 		FROM users u
// 		JOIN user_sessions s ON u.user_id = s.user_id
// 		WHERE s.session_uuid = ? AND s.expires_at > ?`
// 	err := r.db.GetContext(ctx, &userID, query, sessionID, time.Now())
// 	if err != nil {
// 		return 0, err
// 	}
// 	return userID, nil
// }
func (r *SessionRepository) FindUserBySessionID(ctx context.Context, sessionID string) (int, error) {
    var userID int
    // Simplified query using covering index - no JOIN needed
    query := `SELECT user_id FROM user_sessions WHERE session_uuid = ? AND expires_at > ?`
    err := r.db.GetContext(ctx, &userID, query, sessionID, time.Now())
    if err != nil {
        return 0, err
    }
    return userID, nil
}