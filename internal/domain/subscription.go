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

