// Public Domain (-) 2018-present, The Elko Authors.
// See the Elko UNLICENSE file for details.

package stats

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"
	"time"
)

type Histogram struct {
	decay    float64
	horizon  time.Time
	resample time.Duration
	sample   sample
	size     int
	start    time.Time
	values   []int64
}

func (h *Histogram) Len() int {
	return len(h.sample)
}

func (h *Histogram) Mean() float64 {
	if len(h.sample) == 0 {
		return 0
	}
	var total int64
	for _, i := range h.sample {
		total += i.value
	}
	return float64(total) / float64(len(h.sample))
}

func (h *Histogram) Percentile(p float64) float64 {
	slen := len(h.sample)
	if slen == 0 {
		return 0
	}
	if slen == 1 {
		return float64(h.sample[0].value)
	}
	l := h.values[:slen]
	for idx, i := range h.sample {
		l[idx] = i.value
	}
	sort.Sort(values(l))
	pos := p * float64(slen-1)
	ceil := math.Ceil(pos)
	floor := math.Floor(pos)
	if ceil == floor {
		return float64(l[int(pos)])
	}
	return float64(l[int(floor)])*(ceil-pos) + float64(l[int(ceil)])*(pos-floor)
}

func (h *Histogram) StdDev() float64 {
	return math.Sqrt(h.Variance())
}

// Adapted from the various ports of the Coda Hale Metrics library which
// reference the paper "Forward Decay: A Practical Time Decay Model for
// Streaming Systems".
func (h *Histogram) Update(t time.Time, v int64) {
	if len(h.sample) == h.size {
		heap.Pop(&h.sample)
	}
	heap.Push(&h.sample, elem{
		pos:   math.Exp(t.Sub(h.start).Seconds()*h.decay) / rand.Float64(),
		value: v,
	})
	if t.After(h.horizon) {
		diff := math.Exp(t.Sub(h.start).Seconds() * -h.decay)
		h.start = t
		h.horizon = t.Add(h.resample)
		for _, i := range h.sample {
			i.pos *= diff
		}
	}
}

func (h *Histogram) Variance() float64 {
	sample := h.sample
	slen := float64(len(sample))
	if slen == 0 {
		return 0
	}
	var (
		diff  float64
		total int64
	)
	for _, i := range sample {
		total += i.value
	}
	mean := float64(total) / slen
	ft := 0.0
	for _, i := range sample {
		diff = mean - float64(i.value)
		ft += diff * diff
	}
	return ft / slen
}

type elem struct {
	pos   float64
	value int64
}

type sample []elem

func (s sample) Len() int {
	return len(s)
}

func (s sample) Less(i, j int) bool {
	return s[i].pos < s[j].pos
}

func (s *sample) Push(v interface{}) {
	*s = append(*s, v.(elem))
}

func (s sample) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s *sample) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

type values []int64

func (v values) Len() int {
	return len(v)
}

func (v values) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v values) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func New(size int, decay float64, resample time.Duration) *Histogram {
	start := time.Now().UTC()
	return &Histogram{
		decay:    decay,
		horizon:  start.Add(resample),
		resample: resample,
		sample:   make(sample, 0, size),
		size:     size,
		start:    start,
		values:   make([]int64, 0, size),
	}
}
