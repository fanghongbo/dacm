package utils

import (
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"math"
	"runtime"
)

func GetCPULimitNum(maxCPURate float64) int {
	var cpuLimit int

	cpuLimit = int(math.Ceil(float64(runtime.NumCPU()) * maxCPURate))
	if cpuLimit < 1 {
		cpuLimit = 1
	}
	return cpuLimit
}

func CalculateMemLimit(maxMemRate float64) int {
	var (
		memTotal int
		memLimit int
		m        *mem.VirtualMemoryStat
		err      error
	)

	m, err = mem.VirtualMemory()
	if err != nil {
		log.Printf("failed to get virtual memory info: %v\n", err)
		memLimit = 512
	} else {
		memTotal = int(m.Total / (1024 * 1024))
		memLimit = int(float64(memTotal) * maxMemRate)
	}

	if memLimit < 512 {
		memLimit = 512
	}

	return memLimit
}
