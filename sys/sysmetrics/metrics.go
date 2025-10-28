package sysmetrics

import (
	"time"

	psload "github.com/shirou/gopsutil/v4/load"
	psmem "github.com/shirou/gopsutil/v4/mem"
)

type Metrics struct {
	Time time.Time

	Uptime   uint64
	Load     *psload.AvgStat
	LoadMisc *psload.MiscStat
	Mem      *psmem.VirtualMemoryStat
	Net      *NetMetrics
}

type NetMetrics struct {
	Tx uint64
	Rx uint64

	TxAll uint64
	RxAll uint64
}
