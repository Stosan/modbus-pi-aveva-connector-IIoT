package domain

import (
	"aradel-pi/config"
	"aradel-pi/internal/services/omf"
	"aradel-pi/internal/services/piwebapi"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"go.uber.org/zap"
)

// processGateway isolates the connection and reading logic for a single gateway.
// Reconnects gracefully without stalling other gateways.
func (p *Publisher) processOMFGateway(ctx context.Context, gateway config.Gateway,
	omfClient *omf.Client, metrics *GatewayMetrics,
	mode publishMode) {
	p.Logger.Info("📡 Starting Data Collection Loop for Gateway: ", zap.String("gateway", gateway.Address))
	handler := modbus.NewTCPClientHandler(gateway.Address)
	handler.Timeout = 10 * time.Second
	handler.SlaveId = byte(gateway.SlaveID)

	err := handler.Connect()
	if err != nil {
		log.Printf("❌ Modbus TCP Connect failed for %s: %v. Retrying in 10s...", gateway.Address, err)
		sleepWithContext(ctx, 10*time.Second)
	}

	log.Printf("✅ Modbus TCP Connect successful for %s", gateway.Address)

	// Create Modbus client
	client := modbus.NewClient(handler)

	// Single Mutex assigned to protect sequential Modbus queries cleanly
	var mu sync.Mutex

	for {
		if ctx.Err() != nil {
			log.Printf("🛑 Shutting down gateway loop for %s", gateway.Address)
			return
		}

		// Enters polling loop
		err = p.readTagsLoop(ctx, client, gateway, omfClient, nil, &mu, metrics)
		if err != nil {
			log.Printf("⚠️ Polling loop error on %s: %v. Reconnecting...", gateway.Address, err)
		}

		handler.Close()

		if ctx.Err() != nil {
			return
		}

		// Wait before attempting reconnection
		sleepWithContext(ctx, 5*time.Second)
	}
}

// processGateway isolates the connection and reading logic for a single gateway.
// Reconnects gracefully without stalling other gateways.
func (p *Publisher) processPiWebAPIGateway(ctx context.Context,
	gateway config.Gateway, metrics *GatewayMetrics,
	mode publishMode) {
	var (
		handler      modbus.TCPClientHandler
		mu           sync.Mutex
		client       modbus.Client
		piWebService *piwebapi.ServiceClient
	)
	delay := backoffBase
	for {
		// ── Honour shutdown before attempting (re)connect ────────────────────
		select {
		case <-ctx.Done():
			p.Logger.Info("Gateway goroutine exiting (shutdown)")
			return
		default:
		}

		p.Logger.Info("📡 Starting Data Collection Loop for Gateway: ", zap.String("gateway", gateway.Address))
		handler := modbus.NewTCPClientHandler(gateway.Address)
		handler.Timeout = 10 * time.Second
		handler.SlaveId = byte(gateway.SlaveID)

		err := handler.Connect()
		if err != nil {
			metrics.Reconnects.Add(1)
			p.Logger.Error("Modbus connect failed — backing off",
				zap.Error(err),
				zap.Duration("retry_in", delay))
			sleepWithContext(ctx, delay)
			delay = nextBackoff(delay)
			continue
		}

		p.Logger.Info("✅ Modbus TCP Connect successful for ", zap.String("gateway", gateway.Address))
		client = modbus.NewClient(handler)
		piBaseURL := p.PiWebClient.PIServer.BaseURL + "/piwebapi"
		piWebService = piwebapi.NewServiceClient(piBaseURL, p.PiWebClient.PIServer.Username, p.PiWebClient.PIServer.Password)
		break
	}

	for {
		if ctx.Err() != nil {
			p.Logger.Info("🛑 Shutting down gateway loop for ", zap.String("gateway", gateway.Address))
			return
		}

		// Enters polling loop
		err := p.readTagsLoop(ctx, client, gateway, nil, piWebService, &mu, metrics)
		if err != nil {
			p.Logger.Warn("⚠️ Polling loop error on %s: %v. Reconnecting...", zap.String("gateway", gateway.Address), zap.Error(err))
		}

		handler.Close()

		if ctx.Err() != nil {
			return
		}

		// Wait before attempting reconnection
		sleepWithContext(ctx, 5*time.Second)
	}
}

// readTagsLoop performs continuous sweeps over all tags locally for a gateway.
func (p *Publisher) readTagsLoop(ctx context.Context,
	client modbus.Client, gateway config.Gateway,
	omfClient *omf.Client, piWebService *piwebapi.ServiceClient,
	mu *sync.Mutex,
	metrics *GatewayMetrics) error {
	p.Logger.Info("Reading tags for gateway", zap.String("gateway", gateway.Address))
	log := p.Logger.With(zap.String("gateway", gateway.Address))
	tagCount := len(gateway.Tags)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		errorCount := 0

		for _, tag := range gateway.Tags {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			mu.Lock()
			res, err := client.ReadHoldingRegisters(tag.Register, 2)
			mu.Unlock()

			if err != nil {
				metrics.ReadError.Add(1)
				log.Warn("Modbus read error",
					zap.Uint16("register", tag.Register),
					zap.String("device", tag.DeviceType),
					zap.Error(err))
				errorCount++
				time.Sleep(2 * time.Second)
				continue
			}

			// Raw bytes → IEEE 754 float with NaN/Inf guard
			// Validate and convert float data
			bits := binary.BigEndian.Uint32(res)
			sensorValue := math.Float32frombits(bits)
			if math.IsNaN(float64(sensorValue)) || math.IsInf(float64(sensorValue), 0) {
				log.Error("Invalid sensor value (NaN/Inf) — discarding",
					zap.Uint16("register", tag.Register),
					zap.String("device", tag.DeviceType))
				metrics.ReadError.Add(1)
				errorCount++
				continue
			}
			metrics.ReadSuccess.Add(1)
			// Deliver payload upstream
			p.logSensorValue(log, gateway.Address, tag, sensorValue)
			pushErr := p.pushValue(tag, sensorValue, omfClient, piWebService)
			if pushErr != nil {
				metrics.UpstreamPushKO.Add(1)
				log.Error("Upstream push failed",
					zap.String("device", tag.DeviceType),
					zap.Error(pushErr))
			} else {
				metrics.UpstreamPushOK.Add(1)
			}

		}
		// ── All tags failed → assume connection loss; exit for reconnect ─────
		if tagCount > 0 && errorCount >= tagCount {
			return fmt.Errorf("all %d tags failed — assuming Modbus connection loss", tagCount)
		}
		// If all tags failed for this gateway block, it strongly points to connection dropping.
		// Exit to force a reconnect cascade.
		if errorCount >= len(gateway.Tags) {
			log.Warn("Degraded sweep — some tags failed",
				zap.Int("failed", errorCount),
				zap.Int("total", tagCount))
		}

		// Stagger between bulk sweeps per config requirement or arbitrary 10-sec scale
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
		return piWebService.PushValue(tag.PIWebID, value)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// logSensorValue emits a structured log line matching PI AF naming conventions.
// ─────────────────────────────────────────────────────────────────────────────
func (p *Publisher) logSensorValue(log *zap.Logger, gateway string, tag config.Tag, value float32) {
	unit := ""
	if tag.DeviceType == "pressure" {
		unit = "PSI"
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
