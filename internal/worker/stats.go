package worker

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

type Stats struct {
	Memory *mem.VirtualMemoryStat
	Disk   *disk.UsageStat
	Cpu    *cpu.TimesStat
	Load   *load.AvgStat
}

func GetStats() (*Stats, error) {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("getting memory stats: %w", err)
	}
	disk, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("getting disk stats: %w", err)
	}
	cpu, err := cpu.Times(false)
	if err != nil {
		return nil, fmt.Errorf("getting cpu stas: %w", err)
	}
	load, err := load.Avg()
	if err != nil {
		return nil, fmt.Errorf("getting load avg: %w", err)
	}
	return &Stats{
		Memory: mem,
		Disk:   disk,
		Cpu:    &cpu[0],
		Load:   load,
	}, nil
}

func (s *Stats) MemTotalKb() uint64 {
	return s.Memory.Total
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.Memory.Available
}

func (s *Stats) MemUsedKb() uint64 {
	return s.Memory.Used
}

func (s *Stats) MemUsedPercent() uint64 {
	return uint64(s.Memory.UsedPercent)
}

func (s *Stats) DiskTotal() uint64 {
	return s.Disk.Total
}

func (s *Stats) DiskFree() uint64 {
	return s.Disk.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.Disk.Used
}

func (s *Stats) DiskUsedPercent() uint64 {
	return uint64(s.Disk.UsedPercent)
}

func (s *Stats) CpuUsage() float64 {
	idle := s.Cpu.Idle + s.Cpu.Iowait
	nonIdle := s.Cpu.User + s.Cpu.Nice +
		s.Cpu.Steal + s.Cpu.Irq + s.Cpu.Softirq +
		s.Cpu.System + s.Cpu.Guest + s.Cpu.GuestNice
	total := idle + nonIdle

	if total == 0 {
		return 0.00
	}

	return (float64(total) - float64(idle)) / float64(total)
}
