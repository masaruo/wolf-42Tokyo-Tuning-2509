package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
	// "sort"
)

type RobotService struct {
	store *repository.Store
}

func NewRobotService(store *repository.Store) *RobotService {
	return &RobotService{store: store}
}

func (s *RobotService) GenerateDeliveryPlan(ctx context.Context, robotID string, capacity int) (*model.DeliveryPlan, error) {
	var plan model.DeliveryPlan

	err := utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.ExecTx(ctx, func(txStore *repository.Store) error {
			orders, err := txStore.OrderRepo.GetShippingOrders(ctx)
			if err != nil {
				return err
			}
			plan, err = selectOrdersForDelivery(ctx, orders, robotID, capacity)
			if err != nil {
				return err
			}
			if len(plan.Orders) > 0 {
				orderIDs := make([]int64, len(plan.Orders))
				for i, order := range plan.Orders {
					orderIDs[i] = order.OrderID
				}

				if err := txStore.OrderRepo.UpdateStatuses(ctx, orderIDs, "delivering"); err != nil {
					return err
				}
				log.Printf("Updated status to 'delivering' for %d orders", len(orderIDs))
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *RobotService) UpdateOrderStatus(ctx context.Context, orderID int64, newStatus string) error {
	return utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.OrderRepo.UpdateStatuses(ctx, []int64{orderID}, newStatus)
	})
}

// 事前に注文をソートして効率性を向上
// func selectOrdersForDelivery(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
// 	if len(orders) == 0 {
// 		return model.DeliveryPlan{RobotID: robotID}, nil
// 	}

// 	// 価値重量比でソート（貪欲法の改善）
// 	sortedOrders := make([]model.Order, len(orders))
// 	copy(sortedOrders, orders)

// 	// 価値重量比でソート
// 	sort.Slice(sortedOrders, func(i, j int) bool {
// 		ratioI := float64(sortedOrders[i].Value) / float64(sortedOrders[i].Weight)
// 		ratioJ := float64(sortedOrders[j].Value) / float64(sortedOrders[j].Weight)
// 		return ratioI > ratioJ
// 	})

// 	// 残りは既存のDPアルゴリズムまたは改善されたアルゴリズムを使用
// 	n := len(sortedOrders)

// 	// 動的プログラミング用のテーブル
// 	dp := make([][]int, n+1)
// 	for i := range dp {
// 		dp[i] = make([]int, robotCapacity+1)
// 	}

// 	// DPテーブルを埋める
// 	for i := 1; i <= n; i++ {
// 		select {
// 		case <-ctx.Done():
// 			return model.DeliveryPlan{}, ctx.Err()
// 		default:
// 		}

// 		order := sortedOrders[i-1]
// 		for w := 0; w <= robotCapacity; w++ {
// 			// アイテムを選ばない場合
// 			dp[i][w] = dp[i-1][w]

// 			// アイテムを選ぶ場合
// 			if w >= order.Weight {
// 				dp[i][w] = max(dp[i][w], dp[i-1][w-order.Weight]+order.Value)
// 			}
// 		}
// 	}

// 	// 選択されたアイテムを復元
// 	var selectedOrders []model.Order
// 	w := robotCapacity
// 	totalWeight := 0

// 	for i := n; i > 0 && dp[i][w] > 0; i-- {
// 		if dp[i][w] != dp[i-1][w] {
// 			order := sortedOrders[i-1]
// 			selectedOrders = append(selectedOrders, order)
// 			w -= order.Weight
// 			totalWeight += order.Weight
// 		}
// 	}

// 	return model.DeliveryPlan{
// 		RobotID:     robotID,
// 		TotalWeight: totalWeight,
// 		TotalValue:  dp[n][robotCapacity],
// 		Orders:      selectedOrders,
// 	}, nil
// }

// func max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }

func selectOrdersForDelivery(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
	n := len(orders)
	bestValue := 0
	var bestSet []model.Order
	steps := 0
	checkEvery := 16384

	var dfs func(i, curWeight, curValue int, curSet []model.Order) bool
	dfs = func(i, curWeight, curValue int, curSet []model.Order) bool {
		if curWeight > robotCapacity {
			return false
		}
		steps++
		if checkEvery > 0 && steps%checkEvery == 0 {
			select {
			case <-ctx.Done():
				return true
			default:
			}
		}
		if i == n {
			if curValue > bestValue {
				bestValue = curValue
				bestSet = append([]model.Order{}, curSet...)
			}
			return false
		}

		if dfs(i+1, curWeight, curValue, curSet) {
			return true
		}

		order := orders[i]
		return dfs(i+1, curWeight+order.Weight, curValue+order.Value, append(curSet, order))
	}

	canceled := dfs(0, 0, 0, nil)
	if canceled {
		return model.DeliveryPlan{}, ctx.Err()
	}

	var totalWeight int
	for _, o := range bestSet {
		totalWeight += o.Weight
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  bestValue,
		Orders:      bestSet,
	}, nil
}
