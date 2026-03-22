package httpapi

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

func rootDiskPath() string {
	if runtime.GOOS == "windows" {
		if v := os.Getenv("SystemDrive"); v != "" {
			return v + `\`
		}
		return `C:\`
	}
	return "/"
}

// opsHostMetrics returns host CPU, memory, and root filesystem usage via gopsutil
// (https://github.com/shirou/gopsutil). Exposed like /api/ops/summary for operators.
func (h *Handlers) opsHostMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logical, _ := cpu.Counts(true)
	physical, _ := cpu.Counts(false)

	var cpuPct float64
	pcts, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err == nil && len(pcts) > 0 {
		cpuPct = pcts[0]
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	du, err := disk.UsageWithContext(ctx, rootDiskPath())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	cpuBlock := map[string]any{
		"logical_cores":   logical,
		"physical_cores":  physical,
		"percent_total":   cpuPct,
		"sample_interval": "200ms",
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		cpuBlock["load_avg"] = map[string]any{
			"load1":  avg.Load1,
			"load5":  avg.Load5,
			"load15": avg.Load15,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"cpu": cpuBlock,
		"memory": map[string]any{
			"total_bytes":     vm.Total,
			"available_bytes": vm.Available,
			"used_bytes":      vm.Used,
			"used_percent":    vm.UsedPercent,
		},
		"disk": map[string]any{
			"path":          du.Path,
			"total_bytes":   du.Total,
			"free_bytes":    du.Free,
			"used_bytes":    du.Used,
			"used_percent":  du.UsedPercent,
			"inodes_total":  du.InodesTotal,
			"inodes_used":   du.InodesUsed,
			"inodes_free":   du.InodesFree,
			"inodes_percent": du.InodesUsedPercent,
		},
	})
}
