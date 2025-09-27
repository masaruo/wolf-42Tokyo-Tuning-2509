package repository

import (
	"backend/internal/model"
	"context"
)

type ProductRepository struct {
	db DBTX
}

func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

// 商品一覧をデータベース側でページング処理を行う（最適化版）
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// Validate and sanitize sort field to prevent SQL injection
	validSortFields := map[string]bool{
		"product_id":  true,
		"name":        true,
		"value":       true,
		"weight":      true,
		"description": true,
	}

	sortField := "product_id"
	if validSortFields[req.SortField] {
		sortField = req.SortField
	}

	sortOrder := "ASC"
	if req.SortOrder == "DESC" {
		sortOrder = "DESC"
	}

	// Build base queries
	baseQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	`
	countQuery := `SELECT COUNT(*) FROM products`

	args := []interface{}{}
	whereClause := ""

	if req.Search != "" {
		whereClause = " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Get total count first
	var total int
	err := r.db.GetContext(ctx, &total, countQuery+whereClause, args...)
	if err != nil {
		return nil, 0, err
	}

	// Build final query with ORDER BY and LIMIT (database-side pagination)
	finalQuery := baseQuery + whereClause +
		" ORDER BY " + sortField + " " + sortOrder + ", product_id ASC" +
		" LIMIT ? OFFSET ?"

	args = append(args, req.PageSize, req.Offset)

	var products []model.Product
	err = r.db.SelectContext(ctx, &products, finalQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
