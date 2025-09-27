package repository

import (
	"backend/internal/model"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type OrderRepository struct {
	db DBTX
}

func NewOrderRepository(db DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

// 注文を作成し、生成された注文IDを返す
func (r *OrderRepository) Create(ctx context.Context, order *model.Order) (string, error) {
	query := `INSERT INTO orders (user_id, product_id, shipped_status, created_at) VALUES (?, ?, 'shipping', NOW())`
	result, err := r.db.ExecContext(ctx, query, order.UserID, order.ProductID)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// 複数の注文IDのステータスを一括で更新
// 主に配送ロボットが注文を引き受けた際に一括更新をするために使用
func (r *OrderRepository) UpdateStatuses(ctx context.Context, orderIDs []int64, newStatus string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In("UPDATE orders SET shipped_status = ? WHERE order_id IN (?)", newStatus, orderIDs)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// 配送中(shipped_status:shipping)の注文一覧を取得
func (r *OrderRepository) GetShippingOrders(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	query := `
        SELECT
            o.order_id,
            p.weight,
            p.value
        FROM orders o
        JOIN products p ON o.product_id = p.product_id
        WHERE o.shipped_status = 'shipping'
    `
	err := r.db.SelectContext(ctx, &orders, query)
	return orders, err
}

//! mysql> EXPLAIN SELECT o.*, p.name FROM orders o JOIN products p ON o.product_id = p.product_id WHERE o.user_id = 1;
// 2 rows in set, 1 warning (0.001 sec)mysql> EXPLAIN SELECT o.order_id, o.product_id, o.shipped_status, o.created_at, o.arrived_at, p.name as product_name
//     ->         FROM orders o
//     ->         JOIN products p ON o.product_id = p.product_id
//     ->         WHERE o.user_id = 1;
// +----+-------------+-------+------------+--------+--------------------+---------+---------+-----------------------------+------+----------+-------+
// | id | select_type | table | partitions | type   | possible_keys      | key     | key_len | ref                         | rows | filtered | Extra |
// +----+-------------+-------+------------+--------+--------------------+---------+---------+-----------------------------+------+----------+-------+
// |  1 | SIMPLE      | o     | NULL       | ref    | user_id,product_id | user_id | 4       | const                       |    2 |   100.00 | NULL  |
// |  1 | SIMPLE      | p     | NULL       | eq_ref | PRIMARY            | PRIMARY | 4       | 42tokyo2508-db.o.product_id |    1 |   100.00 | NULL  |
// +----+-------------+-------+------------+--------+--------------------+---------+---------+-----------------------------+------+----------+-------+
// 2 rows in set, 1 warning (0.001 sec)

//! mysql> explain SELECT user_id, password_hash, user_name FROM users WHERE user_name = 1;
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra       |
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// |  1 | SIMPLE      | users | NULL       | ALL  | NULL          | NULL | NULL    | NULL |  100 |    10.00 | Using where |
// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------------+
// 1 row in set, 1 warning (0.001 sec)
// 注文履歴一覧を取得
// この結果から、クエリのボトルネックを探します。特に注目すべきカラムは以下の通りです。

// type:
// ALLになっている場合、テーブルをフルスキャンしており、パフォーマンスが悪い可能性があります。
// index、range、ref、eq_ref、constの順に高速になります。ALLは避けたい状態です。

// key:
// 実際に使われたインデックス名が表示されます。NULLの場合、インデックスが使われていないことを意味します。

// rows:
// クエリを実行するためにスキャンが必要だと見積もられた行数です。この値が大きい場合は注意が必要です。

// Extra:
// Using filesortやUsing temporaryと表示されている場合、ソート処理や一時テーブルの作成にコストがかかっており、改善の余地があるかもしれません。

func (r *OrderRepository) ListOrders(ctx context.Context, userID int, req model.ListRequest) ([]model.Order, int, error) {
	// Validate and sanitize sort field to prevent SQL injection
	validSortFields := map[string]bool{
		"order_id":       true,
		"product_name":   true,
		"shipped_status": true,
		"created_at":     true,
		"arrived_at":     true,
	}

	sortField := "o.order_id"
	if validSortFields[req.SortField] {
		switch req.SortField {
		case "product_name":
			sortField = "p.name"
		case "shipped_status":
			sortField = "o.shipped_status"
		case "created_at":
			sortField = "o.created_at"
		case "arrived_at":
			sortField = "o.arrived_at"
		default:
			sortField = "o.order_id"
		}
	}

	sortOrder := "ASC"
	if strings.ToUpper(req.SortOrder) == "DESC" {
		sortOrder = "DESC"
	}

	// Build base queries
	baseQuery := `
		SELECT o.order_id, o.product_id, o.shipped_status, o.created_at, o.arrived_at, p.name as product_name
		FROM orders o
		JOIN products p ON o.product_id = p.product_id
		WHERE o.user_id = ?
	`
	countQuery := `
		SELECT COUNT(*) 
		FROM orders o
		JOIN products p ON o.product_id = p.product_id
		WHERE o.user_id = ?
	`

	args := []interface{}{userID}
	whereClause := ""

	// Add search filter to SQL query instead of application-side filtering
	if req.Search != "" {
		if req.Type == "prefix" {
			whereClause = " AND p.name LIKE ?"
			searchPattern := req.Search + "%"
			args = append(args, searchPattern)
		} else {
			whereClause = " AND p.name LIKE ?"
			searchPattern := "%" + req.Search + "%"
			args = append(args, searchPattern)
		}
	}

	// Get total count first
	var total int
	err := r.db.GetContext(ctx, &total, countQuery+whereClause, args...)
	if err != nil {
		return nil, 0, err
	}

	// Build final query with ORDER BY and LIMIT (database-side pagination)
	finalQuery := baseQuery + whereClause +
		" ORDER BY " + sortField + " " + sortOrder + ", o.order_id ASC" +
		" LIMIT ? OFFSET ?"

	args = append(args, req.PageSize, req.Offset)

	type orderRow struct {
		OrderID       int          `db:"order_id"`
		ProductID     int          `db:"product_id"`
		ShippedStatus string       `db:"shipped_status"`
		CreatedAt     sql.NullTime `db:"created_at"`
		ArrivedAt     sql.NullTime `db:"arrived_at"`
		ProductName   string       `db:"product_name"`
	}

	var ordersRaw []orderRow
	if err := r.db.SelectContext(ctx, &ordersRaw, finalQuery, args...); err != nil {
		return nil, 0, err
	}

	var orders []model.Order
	for _, o := range ordersRaw {
		orders = append(orders, model.Order{
			OrderID:       int64(o.OrderID),
			ProductID:     o.ProductID,
			ProductName:   o.ProductName,
			ShippedStatus: o.ShippedStatus,
			CreatedAt:     o.CreatedAt.Time,
			ArrivedAt:     o.ArrivedAt,
		})
	}

	return orders, total, nil
}

// func (r *OrderRepository) ListOrders(ctx context.Context, userID int, req model.ListRequest) ([]model.Order, int, error) {
// 	query := `
//         SELECT order_id, product_id, shipped_status, created_at, arrived_at
//         FROM orders
//         WHERE user_id = ?
//     `
// 	type orderRow struct {
// 		OrderID       int          `db:"order_id"`
// 		ProductID     int          `db:"product_id"`
// 		ShippedStatus string       `db:"shipped_status"`
// 		CreatedAt     sql.NullTime `db:"created_at"`
// 		ArrivedAt     sql.NullTime `db:"arrived_at"`
// 	}
// 	var ordersRaw []orderRow
// 	if err := r.db.SelectContext(ctx, &ordersRaw, query, userID); err != nil {
// 		return nil, 0, err
// 	}

// 	var orders []model.Order
// 	for _, o := range ordersRaw {
// 		var productName string
// 		if err := r.db.GetContext(ctx, &productName, "SELECT name FROM products WHERE product_id = ?", o.ProductID); err != nil {
// 			return nil, 0, err
// 		}
// 		if req.Search != "" {
// 			if req.Type == "prefix" {
// 				if !strings.HasPrefix(productName, req.Search) {
// 					continue
// 				}
// 			} else {
// 				if !strings.Contains(productName, req.Search) {
// 					continue
// 				}
// 			}
// 		}
// 		orders = append(orders, model.Order{
// 			OrderID:       int64(o.OrderID),
// 			ProductID:     o.ProductID,
// 			ProductName:   productName,
// 			ShippedStatus: o.ShippedStatus,
// 			CreatedAt:     o.CreatedAt.Time,
// 			ArrivedAt:     o.ArrivedAt,
// 		})
// 	}

// 	switch req.SortField {
// 	case "product_name":
// 		if strings.ToUpper(req.SortOrder) == "DESC" {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].ProductName > orders[j].ProductName
// 			})
// 		} else {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].ProductName < orders[j].ProductName
// 			})
// 		}
// 	case "created_at":
// 		if strings.ToUpper(req.SortOrder) == "DESC" {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].CreatedAt.After(orders[j].CreatedAt)
// 			})
// 		} else {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].CreatedAt.Before(orders[j].CreatedAt)
// 			})
// 		}
// 	case "shipped_status":
// 		if strings.ToUpper(req.SortOrder) == "DESC" {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].ShippedStatus > orders[j].ShippedStatus
// 			})
// 		} else {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].ShippedStatus < orders[j].ShippedStatus
// 			})
// 		}
// 	case "arrived_at":
// 		if strings.ToUpper(req.SortOrder) == "DESC" {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				if orders[i].ArrivedAt.Valid && orders[j].ArrivedAt.Valid {
// 					return orders[i].ArrivedAt.Time.After(orders[j].ArrivedAt.Time)
// 				}
// 				return orders[i].ArrivedAt.Valid
// 			})
// 		} else {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				if orders[i].ArrivedAt.Valid && orders[j].ArrivedAt.Valid {
// 					return orders[i].ArrivedAt.Time.Before(orders[j].ArrivedAt.Time)
// 				}
// 				return orders[j].ArrivedAt.Valid
// 			})
// 		}
// 	case "order_id":
// 		fallthrough
// 	default:
// 		if strings.ToUpper(req.SortOrder) == "DESC" {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].OrderID > orders[j].OrderID
// 			})
// 		} else {
// 			sort.SliceStable(orders, func(i, j int) bool {
// 				return orders[i].OrderID < orders[j].OrderID
// 			})
// 		}
// 	}

// 	total := len(orders)
// 	start := req.Offset
// 	end := req.Offset + req.PageSize
// 	if start > total {
// 		start = total
// 	}
// 	if end > total {
// 		end = total
// 	}
// 	pagedOrders := orders[start:end]

// 	return pagedOrders, total, nil
// }
