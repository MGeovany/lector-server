package domain

// StorageLimitBytesForPlan returns the per-user storage quota for a given subscription plan.
//
// We use decimal GB for UX consistency (50GB = 50,000,000,000 bytes).
// Free tier uses MiB (15MB = 15 * 1024 * 1024 bytes).
func StorageLimitBytesForPlan(plan string) int64 {
	switch plan {
	case "pro_monthly", "pro_yearly", "founder_lifetime":
		return 50_000_000_000
	default:
		return 15 * 1024 * 1024
	}
}

// AskAIEnabledForPlan returns whether the Ask AI feature is available for the plan.
func AskAIEnabledForPlan(plan string) bool {
	switch plan {
	case "pro_monthly", "pro_yearly", "founder_lifetime":
		return true
	default:
		return false
	}
}

// MonthlyAITokenLimitForPlan returns the monthly token quota (input+output combined).
// 0 means "no access" (feature disabled).
func MonthlyAITokenLimitForPlan(plan string) int {
	if !AskAIEnabledForPlan(plan) {
		return 0
	}

	// MVP: same quota for Pro and Founder.
	// Keep this conservative; can be made configurable later.
	return 500_000
}

