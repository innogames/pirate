package pirate

import (
	"net"
	"sync"
	"time"
)

const (
	LimiterGcInterval = 5 * time.Minute
)

type LimitInfo struct {
	startedAt time.Time
	count     int
}

type IpLimiter struct {
	max      int
	interval time.Duration
	lookup   map[string]*LimitInfo
	mu       sync.Mutex
}

func NewIpLimiter(max int, interval time.Duration) *IpLimiter {
	limiter := new(IpLimiter)
	limiter.max = max
	limiter.interval = interval
	limiter.lookup = make(map[string]*LimitInfo)

	go limiter.runGcLoop()

	return limiter
}

func (l *IpLimiter) Allow(ip net.IP) bool {
	return l.AllowN(ip, 1)
}

func (l *IpLimiter) AllowN(ip net.IP, n int) bool {
	l.mu.Lock()

	ipStr := ip.String()
	info, ok := l.lookup[ipStr]
	now := time.Now()

	// make sure entry exists
	if !ok {
		if info, ok = l.lookup[ipStr]; !ok {
			info = &LimitInfo{now, 0}
			l.lookup[ipStr] = info
		}
	}

	// initialize current window
	if info.startedAt.Add(l.interval).Before(now) {
		info.startedAt = now
		info.count = 0
	}

	info.count += n
	l.mu.Unlock()

	return info.count <= l.max
}

func (l *IpLimiter) runGcLoop() {
	for {
		time.Sleep(LimiterGcInterval)

		l.mu.Lock()
		clearTime := time.Now().Add(-l.interval)

		for ip, info := range l.lookup {
			if info.startedAt.Before(clearTime) {
				delete(l.lookup, ip)
			}
		}
		l.mu.Unlock()
	}
}

type Limiter struct {
	max      int
	interval time.Duration
	info     *LimitInfo
	mu       sync.Mutex
}

func NewLimiter(max int, interval time.Duration) *Limiter {
	limiter := new(Limiter)
	limiter.max = max
	limiter.interval = interval
	limiter.info = &LimitInfo{time.Unix(0, 0), 0}

	return limiter
}

func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}

func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	now := time.Now()

	if l.info.startedAt.Add(l.interval).Before(now) {
		l.info.startedAt = now
		l.info.count = 0
	}

	l.info.count += n
	l.mu.Unlock()

	return l.info.count <= l.max
}
