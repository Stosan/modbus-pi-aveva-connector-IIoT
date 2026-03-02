package domain

import (
	"aradel-pi/config"
	"aradel-pi/internal/services/omf"
	"aradel-pi/internal/services/piwebapi"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
	"go.uber.org/zap"
)

// processGateway isolates the connection and reading logic for a single gateway.
// Reconnects gracefully without stalling other gateways.
func (p *Publisher) processOMFGateway(ctx context.Context, gateway config.Gateway,
	 metrics *GatewayMetrics,
	mode publishMode) {
    const poolSize = 4 // tune to your Modbus slave's concurrent connection limit
    delay := backoffBase

  for {
        select {
        case <-ctx.Done():
            p.Logger.Info("Gateway goroutine exiting", zap.String("gateway", gateway.Address))
            return
        default:
        }

        pool, err := newModbusPool(gateway.Address, byte(gateway.SlaveID), poolSize)
        if err != nil {
            metrics.Reconnects.Add(1)
            p.Logger.Error("Modbus pool init failed",
                zap.String("gateway", gateway.Address),
                zap.Error(err),
                zap.Duration("retry_in", delay))
            sleepWithContext(ctx, delay)
            delay = nextBackoff(delay)
            continue
        }
        delay = backoffBase // reset on success

        piBaseURL := p.OMFClient.PIServer.BaseURL + "/piwebapi/omf"
        omfClient := omf.NewClient(
            piBaseURL,
            p.OMFClient.PIServer.Username,
            p.OMFClient.PIServer.Password,
        )
		if p.Debug {
			p.Logger.Info("✅ Modbus pool ready", zap.String("gateway", gateway.Address))
		}

        err = p.readTagsLoop(ctx, pool, gateway, omfClient, nil, metrics)
        if err != nil {
            p.Logger.Warn("Polling loop exited — will reconnect",
                zap.String("gateway", gateway.Address),
                zap.Error(err))
        }

        if ctx.Err() != nil {
            return
        }
        sleepWithContext(ctx, 5*time.Second)
    }
}


// processGateway isolates the connection and reading logic for a single gateway.
// Reconnects gracefully without stalling other gateways.
func (p *Publisher) processPiWebAPIGateway(
    ctx context.Context,
    gateway config.Gateway,
    metrics *GatewayMetrics,
    mode publishMode,
) {
    const poolSize = 4 // tune to your Modbus slave's concurrent connection limit
    delay := backoffBase

    if !p.Debug{
        gateway.Address = gateway.LocalAddress
    }

    for {
        select {
        case <-ctx.Done():
            p.Logger.Info("Gateway goroutine exiting", zap.String("gateway", gateway.Address))
            return
        default:
        }

        pool, err := newModbusPool(gateway.Address, byte(gateway.SlaveID), poolSize)
        if err != nil {
            metrics.Reconnects.Add(1)
            p.Logger.Error("Modbus pool init failed",
                zap.String("gateway", gateway.Address),
                zap.Error(err),
                zap.Duration("retry_in", delay))
            sleepWithContext(ctx, delay)
            delay = nextBackoff(delay)
            continue
        }
        delay = backoffBase // reset on success

        piBaseURL := p.PiWebClient.PIServer.BaseURL + "/piwebapi"
        piWebService := piwebapi.NewServiceClient(
            piBaseURL,
            p.PiWebClient.PIServer.Username,
            p.PiWebClient.PIServer.Password,
        )
		if p.Debug {
			p.Logger.Info("✅ Modbus pool ready", zap.String("gateway", gateway.Address))
		}

        err = p.readTagsLoop(ctx, pool, gateway, nil, piWebService, metrics)
        if err != nil {
            p.Logger.Warn("Polling loop exited — will reconnect",
                zap.String("gateway", gateway.Address),
                zap.Error(err))
        }

        if ctx.Err() != nil {
            return
        }
        sleepWithContext(ctx, 5*time.Second)
    }
}

// readTagsLoop performs continuous sweeps over all tags locally for a gateway.
func (p *Publisher) readTagsLoop(ctx context.Context,
	pool *modbusPool, 
	gateway config.Gateway, 
	omfClient *omf.Client, 
	piWebService *piwebapi.ServiceClient,
	metrics *GatewayMetrics) error {
    log := p.Logger.With(zap.String("gateway", gateway.Address))
    tagCount := len(gateway.Tags)

    for {
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        var (
            wg           sync.WaitGroup
            errCount     atomic.Int32
            successCount atomic.Int32
            sem          = make(chan struct{}, 4) // bound concurrency to pool size
        )

        for _, tag := range gateway.Tags {
            sem <- struct{}{} // acquire slot before spawning
            wg.Add(1)
            go func() {
                defer wg.Done()
                defer func() { <-sem }() // release slot

                // Acquire a connection from the pool (non-blocking with ctx)
                conn, err := pool.Acquire(ctx)
                if err != nil {
                    log.Warn("Failed to acquire Modbus connection", zap.Error(err))
                    errCount.Add(1)
                    return
                }
                defer pool.Release(conn)

                res, err := conn.ReadHoldingRegisters(tag.Register, 2)
                if err != nil {
                    metrics.ReadError.Add(1)
                    log.Warn("Modbus read error",
                        zap.Uint16("register", tag.Register),
                        zap.String("device", tag.DeviceType),
                        zap.Error(err))
                    errCount.Add(1)
                    time.Sleep(2 * time.Second)
                    return
                }

                bits := binary.BigEndian.Uint32(res)
                sensorValue := math.Float32frombits(bits)
                if math.IsNaN(float64(sensorValue)) || math.IsInf(float64(sensorValue), 0) {
                    log.Error("Invalid sensor value (NaN/Inf) — discarding",
                        zap.Uint16("register", tag.Register),
                        zap.String("device", tag.DeviceType))
                    metrics.ReadError.Add(1)
                    errCount.Add(1)
                    return
                }

                metrics.ReadSuccess.Add(1)
                successCount.Add(1)
				if p.Debug {
					p.logSensorValue(log, gateway.Address, tag, sensorValue)
				}

                if pushErr := piWebService.PushValue(tag.PIWebID, sensorValue, p.Debug); pushErr != nil {
                    metrics.UpstreamPushKO.Add(1)
                    log.Error("Upstream push failed",
                        zap.String("device", tag.DeviceType),
                        zap.Error(pushErr))
                } else {
                    metrics.UpstreamPushOK.Add(1)
                }
            }()
        }

        wg.Wait() // All tags in this sweep complete before sleeping

        failed := int(errCount.Load())
        if tagCount > 0 && failed >= tagCount {
            return fmt.Errorf("all %d tags failed — assuming Modbus connection loss", tagCount)
        }
        if failed > 0 {
            log.Warn("Degraded sweep", zap.Int("failed", failed), zap.Int("total", tagCount))
        }

        sleepWithContext(ctx, sweepInterval)
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// pushValue routes the sensor reading to the correct upstream target.
// Returns an error so the caller can track push failures without panicking.
// ─────────────────────────────────────────────────────────────────────────────
func (p *Publisher) pushValue(
	tag config.Tag,
	value float32,
	omfClient *omf.Client,
	piWebService *piwebapi.ServiceClient,
) error {
	if omfClient != nil {
		return omfClient.SendOMFData(tag.OMFContainerID, value)
	}
	if piWebService != nil {
		return piWebService.PushValue(tag.PIWebID, value, p.Debug)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// logSensorValue emits a structured log line matching PI AF naming conventions.
// ─────────────────────────────────────────────────────────────────────────────
func (p *Publisher) logSensorValue(log *zap.Logger, gateway string, tag config.Tag, value float32) {
	unit := ""
	switch tag.DeviceType {
	case "pressure":
		unit = "PSI"
	case "temperature":
		unit = "°C"
	}
	log.Info("Sensor read",
		zap.String("gateway", gateway),
		zap.String("device_type", tag.DeviceType),
		zap.Uint16("register", tag.Register),
		zap.Float32("value", value),
		zap.String("unit", unit),
	)
}

// ─────────────────────────────────────────────────────────────────────────────
// sleepWithContext sleeps for d but returns early if ctx is cancelled.
// Prevents goroutines from being stuck in a backoff sleep during shutdown.
// ─────────────────────────────────────────────────────────────────────────────
func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// nextBackoff computes the next exponential backoff capped at backoffMax.
func nextBackoff(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * backoffFactor)
	if next > backoffMax {
		return backoffMax
	}
	return next
}
