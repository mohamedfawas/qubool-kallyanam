package config

import (
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/constants"
)

// SubscriptionPlan represents a subscription plan configuration
type SubscriptionPlan struct {
	ID           string   `json:"id" yaml:"id"`
	Name         string   `json:"name" yaml:"name"`
	DurationDays int      `json:"duration_days" yaml:"duration_days"`
	Amount       float64  `json:"amount" yaml:"amount"` // in rupees
	Currency     string   `json:"currency" yaml:"currency"`
	Description  string   `json:"description" yaml:"description"`
	Features     []string `json:"features" yaml:"features"`
	IsActive     bool     `json:"is_active" yaml:"is_active"`
}

// PlansConfig holds all subscription plans
type PlansConfig struct {
	Available map[string]SubscriptionPlan `json:"available" yaml:"available"`
}

// GetDefaultPlansConfig returns the default plans configuration for MVP
func GetDefaultPlansConfig() *PlansConfig {
	return &PlansConfig{
		Available: map[string]SubscriptionPlan{
			constants.DefaultPlanID: {
				ID:           constants.DefaultPlanID,
				Name:         constants.DefaultPlanName,
				DurationDays: constants.DefaultPlanDurationDays,
				Amount:       constants.DefaultPlanAmount,
				Currency:     constants.DefaultCurrency,
				Description:  "Premium membership with full access to all features",
				Features: []string{
					"Unlimited profile views",
					"Chat with all members",
					"Advanced search filters",
					"Priority customer support",
					"Access to exclusive events",
				},
				IsActive: true,
			},
		},
	}
}

// GetPlan returns a plan by ID
func (pc *PlansConfig) GetPlan(planID string) (SubscriptionPlan, bool) {
	plan, exists := pc.Available[planID]
	return plan, exists
}

// GetActivePlans returns all active plans
func (pc *PlansConfig) GetActivePlans() map[string]SubscriptionPlan {
	activePlans := make(map[string]SubscriptionPlan)
	for id, plan := range pc.Available {
		if plan.IsActive {
			activePlans[id] = plan
		}
	}
	return activePlans
}
