package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
	"sort"
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
			
			log.Printf("Robot %s: found %d orders with status 'shipping'", robotID, len(orders))
			
			plan, err = selectOrdersForDeliveryOptimized(ctx, orders, robotID, capacity)
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
				log.Printf("Robot %s: knapsack selected %d orders (total weight: %d, total value: %d)", 
					robotID, len(orderIDs), plan.TotalWeight, plan.TotalValue)
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

// Optimized knapsack using Dynamic Programming with space optimization - O(n*capacity)
func selectOrdersForDeliveryOptimized(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
	n := len(orders)
	if n == 0 {
		return model.DeliveryPlan{
			RobotID:     robotID,
			TotalWeight: 0,
			TotalValue:  0,
			Orders:      []model.Order{},
		}, nil
	}

	// Use greedy for very large datasets to avoid memory issues
	if n > 300 || robotCapacity > 3000 {
		return selectOrdersGreedy(orders, robotID, robotCapacity), nil
	}

	// Space-optimized DP solution - O(n * capacity) time, O(capacity) space
	dp := make([]int, robotCapacity+1)
	parent := make([][]int, n+1)
	for i := range parent {
		parent[i] = make([]int, robotCapacity+1)
	}

	// Fill DP table with backtracking information
	for i := 1; i <= n; i++ {
		order := orders[i-1]
		// Check context cancellation periodically
		if i%50 == 0 {
			select {
			case <-ctx.Done():
				return model.DeliveryPlan{}, ctx.Err()
			default:
			}
		}

		// Process in reverse order to avoid overwriting
		for w := robotCapacity; w >= order.Weight; w-- {
			if dp[w-order.Weight]+order.Value > dp[w] {
				dp[w] = dp[w-order.Weight] + order.Value
				parent[i][w] = 1 // Mark as selected
			}
		}
	}

	// Backtrack to find selected orders
	selectedOrders := make([]model.Order, 0)
	totalWeight := 0
	w := robotCapacity
	
	for i := n; i > 0 && w > 0; i-- {
		if parent[i][w] == 1 {
			order := orders[i-1]
			selectedOrders = append(selectedOrders, order)
			w -= order.Weight
			totalWeight += order.Weight
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  dp[robotCapacity],
		Orders:      selectedOrders,
	}, nil
}

// Greedy approach for very large datasets - O(n log n) with optimized sorting
func selectOrdersGreedy(orders []model.Order, robotID string, robotCapacity int) model.DeliveryPlan {
	if len(orders) == 0 {
		return model.DeliveryPlan{
			RobotID:     robotID,
			TotalWeight: 0,
			TotalValue:  0,
			Orders:      []model.Order{},
		}
	}

	// Pre-allocate slice for better performance
	selectedOrders := make([]model.Order, 0, min(len(orders), robotCapacity/10)) // Estimate capacity
	totalWeight := 0
	totalValue := 0

	// Use in-place sorting for better memory efficiency
	sort.Slice(orders, func(i, j int) bool {
		// Sort by value/weight ratio (greedy heuristic)
		// Avoid division by zero
		if orders[i].Weight == 0 {
			return false
		}
		if orders[j].Weight == 0 {
			return true
		}
		ratio1 := float64(orders[i].Value) / float64(orders[i].Weight)
		ratio2 := float64(orders[j].Value) / float64(orders[j].Weight)
		return ratio1 > ratio2
	})

	for _, order := range orders {
		if totalWeight+order.Weight <= robotCapacity {
			selectedOrders = append(selectedOrders, order)
			totalWeight += order.Weight
			totalValue += order.Value
		}
		// Early termination if we can't fit any more items
		if totalWeight >= int(float64(robotCapacity)*0.95) { // 95% capacity threshold
			break
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  totalValue,
		Orders:      selectedOrders,
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}