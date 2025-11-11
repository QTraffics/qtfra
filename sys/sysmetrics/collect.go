package sysmetrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/qtraffics/qtfra/enhancements/contextlib"
	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/log"
	"github.com/qtraffics/qtfra/sys/sysvars"
	"github.com/qtraffics/qtfra/threads"
	"github.com/qtraffics/qtfra/values"

	pshost "github.com/shirou/gopsutil/v4/host"
	psload "github.com/shirou/gopsutil/v4/load"
	psmem "github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
)

var (
	errCollectorCloseSuccess = ex.New("closed")
	errSubscriberClosed      = ex.New("subscribe channel closed")
)

const (
	defaultChannelQueue = 16
	defaultTopic        = 0
)

type collectTask struct {
	ctx    context.Context
	m      *Metrics
	access sync.Mutex
	wg     *sync.WaitGroup
	err    error
}

type Collector struct {
	logger log.Logger
	hub    *threads.SubHub[*collectTask, int]

	runOnce sync.Once
	done    chan struct{}
}

func NewCollector(logger log.Logger) *Collector {
	return &Collector{
		logger: values.UseDefaultNil(logger, log.Logger(log.Default())),
	}
}

func (c *Collector) Collect(ctx context.Context) (*Metrics, error) {
	ct := c.newCollectTask(ctx)
	c.hub.Publish(defaultTopic, ct)
	ct.wg.Wait()

	return ct.m, ct.err
}

func (c *Collector) newCollectTask(ctx context.Context) *collectTask {
	ct := &collectTask{}
	ct.ctx = ctx
	ct.m = &Metrics{Time: time.Now()}
	ct.wg = &sync.WaitGroup{}
	return ct
}

func (c *Collector) addProducer(g *threads.Group, name string, fn func(ct *collectTask) error) {
	g.Append(name, func(ctx context.Context) error {
		subscribe := c.hub.Subscribe(defaultTopic, defaultChannelQueue)
		defer subscribe.Unsubscribe()
		logger := log.With(c.logger, slog.String("collector", name))
		for {
			select {
			case <-c.done:
				return errCollectorCloseSuccess
			case <-ctx.Done():
				return ctx.Err()
			case ct, ok := <-subscribe.Channel():
				if !ok {
					if ct.wg != nil {
						ct.wg.Done()
					}
					return errSubscriberClosed
				}

				if contextlib.Done(ct.ctx) {
					logger.Debug("collect task has been canceled, skip")
					if ct.wg != nil {
						ct.wg.Done()
					}
					continue
				}
				e := fn(ct)
				if e != nil {
					if sysvars.DebugEnabled {
						logger.Error("collect system metrics failed", log.AttrError(e))
					}
					ct.err = e
				}
				ct.wg.Done()
			}
		}
	})
}

func (c *Collector) Start(ctx context.Context) error {
	c.runOnce.Do(func() {
		if c.hub == nil {
			c.hub = &threads.SubHub[*collectTask, int]{}
		}
		var group threads.Group
		logger := c.logger

		c.addProducer(&group, "net", func(ct *collectTask) error {
			counter, err := psnet.IOCounters(false)
			if err != nil {
				return err
			}
			if len(counter) > 0 && err == nil {
				ct.access.Lock()

				m := ct.m
				m.Net = new(NetMetrics)
				m.Net.TxAll = counter[0].BytesSent
				m.Net.RxAll = counter[0].BytesRecv

				ct.access.Unlock()
			}
			return nil
		})

		c.addProducer(&group, "loadAvg", func(ct *collectTask) error {
			loadAvg, err := psload.Avg()
			if err != nil {
				return err
			}
			if loadAvg != nil && err == nil {
				ct.access.Lock()
				m := ct.m
				m.Load = loadAvg
				ct.access.Unlock()
			}
			return nil
		})

		c.addProducer(&group, "loadMisc", func(ct *collectTask) error {
			loadMisc, err := psload.Misc()
			if err != nil {
				return err
			}

			if loadMisc != nil && err == nil {
				ct.access.Lock()
				m := ct.m
				m.LoadMisc = loadMisc
				ct.access.Unlock()
			}
			return nil
		})

		c.addProducer(&group, "mem", func(ct *collectTask) error {
			vmStat, err := psmem.VirtualMemory()
			if err != nil {
				return err
			}
			if vmStat != nil && err == nil {
				ct.access.Lock()
				m := ct.m
				m.Mem = vmStat
				ct.access.Unlock()
			}
			return nil
		})

		c.addProducer(&group, "uptime", func(ct *collectTask) error {
			uptime, err := pshost.Uptime()
			if err != nil {
				return err
			}
			if uptime != 0 && err == nil {
				ct.access.Lock()
				m := ct.m
				m.Uptime = uptime
				ct.access.Unlock()
			}
			return nil
		})

		go func() {
			c.done = make(chan struct{})
			group.FastFail()
			err := group.Run(ctx)
			logger.Debug("collector quited")
			if err != nil {
				if ex.IsMulti(err, context.Canceled, errCollectorCloseSuccess) {
					close(c.done)
					return
				}
				if ex.IsMulti(err, errSubscriberClosed) {
					panic("subscriber closed unexcepted")
				}
				logger.Error("collector quit unexcepted", log.AttrError(err))
			}
			close(c.done)
		}()
	})
	return nil
}

func (c *Collector) Close() error {
	if c.done == nil {
		return nil
	}
	select {
	case _, done := <-c.done:
		if !done {
			close(c.done)
			return nil
		}
		return ex.New("collector has been closed")
	default:
		close(c.done)
		return nil
	}
}

//func Collect(ctx context.Context) (*Metrics, error) {
//	var (
//		group threads.Group
//		m     = new(Metrics)
//	)
//
//	group.Append("net", func(ctx context.Context) error {
//		counter, err := psnet.IOCounters(false)
//		if err != nil {
//			return err
//		}
//		if len(counter) > 0 {
//			m.Net = new(NetMetrics)
//
//			m.Net.TxAll = counter[0].BytesSent
//			m.Net.RxAll = counter[0].BytesRecv
//		}
//		return nil
//	})
//	group.Append("load", func(ctx context.Context) error {
//		loadAvg, err := psload.Avg()
//		if err != nil {
//			return err
//		}
//		if loadAvg != nil {
//			m.Load = loadAvg
//		}
//
//		loadMisc, err := psload.Misc()
//		if err != nil {
//			return err
//		}
//		if loadMisc != nil {
//			m.LoadMisc = loadMisc
//		}
//		return nil
//	})
//	group.Append("memory", func(ctx context.Context) error {
//		vmStat, err := psmem.VirtualMemory()
//		if err != nil {
//			return err
//		}
//		if vmStat != nil {
//			m.Mem = vmStat
//		}
//		return nil
//	})
//	group.Append("uptime", func(ctx context.Context) error {
//		uptime, err := pshost.Uptime()
//		if err != nil {
//			return err
//		}
//		m.Uptime = uptime
//		return nil
//	})
//
//	err := group.Run(ctx)
//	m.Time = time.Now()
//	if err != nil {
//		return nil, err
//	}
//	return m, nil
//}
