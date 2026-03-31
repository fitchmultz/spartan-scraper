// Purpose: Load and validate retention-policy startup configuration in one focused module.
// Responsibilities:
// - Parse RETENTION_* environment variables into Config.
// - Enforce retention invariants and surface operator-facing startup notices for non-fatal issues.
// - Detect explicit retention-limit overrides so disabled retention remains legible to operators.
// Scope:
// - Retention configuration only; runtime retention policy types remain in retention.go.
// Usage:
// - Call loadRetentionConfig during Load(), then run validateAndFixRetentionConfig.
// Invariants/Assumptions:
// - Validation preserves fail-open behavior by correcting invalid values instead of aborting startup.
// - Explicit override detection ignores unchanged default values.
package config

func loadRetentionConfig(cfg Config) Config {
	cfg.RetentionEnabled = getenvBool("RETENTION_ENABLED", false)
	cfg.RetentionJobDays = getenvInt("RETENTION_JOB_DAYS", 30)
	cfg.RetentionCrawlStateDays = getenvInt("RETENTION_CRAWL_STATE_DAYS", 90)
	cfg.RetentionMaxJobs = getenvInt("RETENTION_MAX_JOBS", 10000)
	cfg.RetentionMaxStorageGB = getenvInt("RETENTION_MAX_STORAGE_GB", 10)
	cfg.RetentionCleanupIntervalHours = getenvInt("RETENTION_CLEANUP_INTERVAL_HOURS", 24)
	cfg.RetentionDryRunDefault = getenvBool("RETENTION_DRY_RUN_DEFAULT", false)
	return cfg
}

// validateAndFixRetentionConfig ensures retention configuration invariants.
// It records operator-visible notices and applies sensible defaults for invalid configurations.
func validateAndFixRetentionConfig(cfg Config) Config {
	if cfg.RetentionJobDays < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-job-days-invalid",
			Severity: "warning",
			Title:    "Job retention age was reset",
			Message:  "RETENTION_JOB_DAYS must be non-negative, so Spartan is using unlimited retention for this session.",
		})
		cfg.RetentionJobDays = 0
	}
	if cfg.RetentionCrawlStateDays < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-crawl-days-invalid",
			Severity: "warning",
			Title:    "Crawl-state retention age was reset",
			Message:  "RETENTION_CRAWL_STATE_DAYS must be non-negative, so Spartan is using unlimited retention for this session.",
		})
		cfg.RetentionCrawlStateDays = 0
	}
	if cfg.RetentionMaxJobs < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-max-jobs-invalid",
			Severity: "warning",
			Title:    "Retention max jobs was reset",
			Message:  "RETENTION_MAX_JOBS must be non-negative, so Spartan is using unlimited jobs for this session.",
		})
		cfg.RetentionMaxJobs = 0
	}
	if cfg.RetentionMaxStorageGB < 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-max-storage-invalid",
			Severity: "warning",
			Title:    "Retention max storage was reset",
			Message:  "RETENTION_MAX_STORAGE_GB must be non-negative, so Spartan is using unlimited storage for this session.",
		})
		cfg.RetentionMaxStorageGB = 0
	}
	if cfg.RetentionCleanupIntervalHours <= 0 {
		recordStartupNotice(StartupNotice{
			ID:       "retention-cleanup-interval-invalid",
			Severity: "warning",
			Title:    "Retention cleanup interval was reset",
			Message:  "RETENTION_CLEANUP_INTERVAL_HOURS must be positive, so Spartan is using 24 hours for this session.",
		})
		cfg.RetentionCleanupIntervalHours = 24
	}

	if !cfg.RetentionEnabled {
		hasLimits := cfg.RetentionJobDays > 0 || cfg.RetentionMaxJobs > 0 || cfg.RetentionMaxStorageGB > 0
		if hasLimits && hasExplicitRetentionLimitOverrides() {
			recordStartupNotice(StartupNotice{
				ID:       "retention-disabled-with-limits",
				Severity: "warning",
				Title:    "Retention limits are configured but inactive",
				Message:  "Retention limits are set while RETENTION_ENABLED is false, so automatic cleanup will not run until retention is enabled.",
			})
		}
	}

	return cfg
}

func hasExplicitRetentionLimitOverrides() bool {
	defaults := map[string]string{
		"RETENTION_JOB_DAYS":         "30",
		"RETENTION_CRAWL_STATE_DAYS": "90",
		"RETENTION_MAX_JOBS":         "10000",
		"RETENTION_MAX_STORAGE_GB":   "10",
	}

	for key, defaultValue := range defaults {
		value, ok := lookupEnvNormalized(key)
		if !ok {
			continue
		}
		if value != "" && value != defaultValue {
			return true
		}
	}

	return false
}
