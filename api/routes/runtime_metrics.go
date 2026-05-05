package routes

import (
	"runtime"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SetupRuntimeMetricsRoute exposes a small runtime snapshot for operational debugging.
func SetupRuntimeMetricsRoute(app *fiber.App, dbClient *gorm.DB) {
	app.Get("/indexer/debug/runtime", func(c *fiber.Ctx) error {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		response := fiber.Map{
			"go": fiber.Map{
				"goroutines": runtime.NumGoroutine(),
				"memory": fiber.Map{
					"alloc_bytes":         mem.Alloc,
					"total_alloc_bytes":   mem.TotalAlloc,
					"sys_bytes":           mem.Sys,
					"heap_alloc_bytes":    mem.HeapAlloc,
					"heap_sys_bytes":      mem.HeapSys,
					"heap_idle_bytes":     mem.HeapIdle,
					"heap_in_use_bytes":   mem.HeapInuse,
					"heap_released_bytes": mem.HeapReleased,
					"heap_objects":        mem.HeapObjects,
					"next_gc_bytes":       mem.NextGC,
				},
				"gc": fiber.Map{
					"count":                   mem.NumGC,
					"pause_total_ns":          mem.PauseTotalNs,
					"last_gc_unix_nano":       mem.LastGC,
					"gc_cpu_fraction":         mem.GCCPUFraction,
					"forced_gc_count":         mem.NumForcedGC,
					"gc_sys_bytes":            mem.GCSys,
					"stack_in_use_bytes":      mem.StackInuse,
					"stack_sys_bytes":         mem.StackSys,
					"mspan_in_use_bytes":      mem.MSpanInuse,
					"mspan_sys_bytes":         mem.MSpanSys,
					"mcache_in_use_bytes":     mem.MCacheInuse,
					"mcache_sys_bytes":        mem.MCacheSys,
					"buck_hash_sys_bytes":     mem.BuckHashSys,
					"gc_metadata_sys_bytes":   mem.GCSys,
					"other_runtime_sys_bytes": mem.OtherSys,
				},
			},
		}

		if sqlDB, err := dbClient.DB(); err == nil {
			stats := sqlDB.Stats()
			response["database"] = fiber.Map{
				"max_open_connections": stats.MaxOpenConnections,
				"open_connections":     stats.OpenConnections,
				"in_use":               stats.InUse,
				"idle":                 stats.Idle,
				"wait_count":           stats.WaitCount,
				"wait_duration_ns":     stats.WaitDuration.Nanoseconds(),
				"max_idle_closed":      stats.MaxIdleClosed,
				"max_idle_time_closed": stats.MaxIdleTimeClosed,
				"max_lifetime_closed":  stats.MaxLifetimeClosed,
			}
		}

		return c.JSON(response)
	})
}
