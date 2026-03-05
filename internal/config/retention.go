// Package config provides data retention policy types and configuration.
//
// This file defines retention policy structures that control how long
// jobs and crawl states are kept before automatic cleanup.
//
// Policies can be configured globally or per-job-kind (scrape, crawl, research).
// A value of 0 means "unlimited" (no retention limit for that dimension).
package config

// RetentionPolicy represents a single retention rule.
// All limits are applied with OR logic - if ANY limit is exceeded,
// the job becomes eligible for cleanup.
type RetentionPolicy struct {
	MaxAgeDays   int // Maximum age in days (0 = unlimited)
	MaxCount     int // Maximum number of jobs to keep (0 = unlimited)
	MaxStorageGB int // Maximum storage in GB (0 = unlimited)
}

// IsEnabled returns true if any retention limit is set.
func (p RetentionPolicy) IsEnabled() bool {
	return p.MaxAgeDays > 0 || p.MaxCount > 0 || p.MaxStorageGB > 0
}

// RetentionPolicies holds policies by job kind.
// Nil values indicate "use default policy".
type RetentionPolicies struct {
	Default  RetentionPolicy
	Scrape   *RetentionPolicy // nil = use default
	Crawl    *RetentionPolicy // nil = use default
	Research *RetentionPolicy // nil = use default
}

// GetPolicy returns the policy for a specific job kind.
// Falls back to Default if no specific policy is configured.
func (p RetentionPolicies) GetPolicy(kind string) RetentionPolicy {
	switch kind {
	case "scrape":
		if p.Scrape != nil {
			return *p.Scrape
		}
	case "crawl":
		if p.Crawl != nil {
			return *p.Crawl
		}
	case "research":
		if p.Research != nil {
			return *p.Research
		}
	}
	return p.Default
}

// RetentionStatus holds the current state of the retention system.
type RetentionStatus struct {
	Enabled              bool   // Whether retention is enabled
	JobRetentionDays     int    // Current job retention setting
	CrawlStateDays       int    // Current crawl state retention setting
	MaxJobs              int    // Current max jobs limit
	MaxStorageGB         int    // Current max storage limit
	TotalJobs            int64  // Current total job count
	JobsEligible         int64  // Jobs eligible for cleanup
	StorageUsedMB        int64  // Current storage usage
	LastCleanup          *int64 // Unix timestamp of last cleanup (nil if never)
	NextScheduledCleanup *int64 // Unix timestamp of next scheduled cleanup
}
