// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package capacity

import (
	"runtime"
	"time"

	"github.com/tav/elko/pkg/stats"
	"github.com/tav/golly/log"
)

type Hook func(cpu float64, mem float64, timestamp time.Time)

// TODO(tav): Sanity check floating point usage for accuracy.
func Monitor(percentile float64, interval time.Duration, hook Hook) {

	cpuHist := stats.New(3000, 0.015, time.Hour)
	memHist := stats.New(3000, 0.015, time.Hour)
	numCPU := int64(runtime.NumCPU())

	var (
		cpu float64
		err error
		mem float64
		now time.Time
		v   float64
	)

	for {
		now = time.Now().UTC()
		v, err = cpuinfo()
		if err == nil {
			cpuHist.Update(now, int64(v*1000000000)/numCPU)
			cpu = cpuHist.Percentile(percentile) / 1000000000
		} else {
			log.Errorf("capacity: could not process cpu info: %s", err)
		}
		v, err = meminfo()
		if err == nil {
			memHist.Update(now, int64(v*1000000000))
			mem = memHist.Percentile(percentile) / 1000000000
		} else {
			log.Errorf("capacity: could not process mem info: %s", err)
		}
		hook(cpu, mem, now)
		time.Sleep(interval)
	}

}
