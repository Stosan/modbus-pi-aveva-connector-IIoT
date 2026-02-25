package domain

import (
	"aradel-pi/config"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// ─────────────────────────────────────────────────────────────────────────────
// publishMode distinguishes the upstream push target.
// ─────────────────────────────────────────────────────────────────────────────
type publishMode int

const (
	modePiWebAPI publishMode = iota
	modeOMF
)


// ─────────────────────────────────────────────────────────────────────────────
// Backoff configuration – Aveva/OT networks benefit from bounded exponential
// backoff to avoid log storms and PLC overload during comms outages.
// ─────────────────────────────────────────────────────────────────────────────
const (
	backoffBase    = 2 * time.Second
	backoffMax     = 120 * time.Second
	backoffFactor  = 2.0
	sweepInterval  = 10 * time.Second
	connectTimeout = 10 * time.Second
	readTimeout    = 5 * time.Second
)

// ─────────────────────────────────────────────────────────────────────────────
// GatewayMetrics holds per-gateway operational counters (atomic for thread
// safety). These can be scraped by Prometheus/PI Asset Framework.
// ─────────────────────────────────────────────────────────────────────────────
type GatewayMetrics struct {
	ReadSuccess    atomic.Int64
	ReadError      atomic.Int64
	UpstreamPushOK atomic.Int64
	UpstreamPushKO atomic.Int64
	Reconnects     atomic.Int64
}

type Publisher struct {
	OMFClient   *config.Config
	PiWebClient *config.Config
	Logger      *zap.Logger
}
