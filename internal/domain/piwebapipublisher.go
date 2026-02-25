package domain

import (
	"context"
	"sync"
	"go.uber.org/zap"
)

func (p *Publisher) StartPiWebAPIPublisher(ctx context.Context) {
	p.Logger.Info("Pi Web API Service initiated...")

	if p.PiWebClient == nil || len(p.PiWebClient.Gateways) == 0 {
		p.Logger.Warn("No Pi Web API gateways configured.")
		return
	}

	var wg sync.WaitGroup

	// Launch a dedicated goroutine per Gateway for true concurrency and scalability.
	for _, gatewayInfo := range p.PiWebClient.Gateways {
		gw := gatewayInfo
		if len(gw.Tags) == 0 {
			p.Logger.Warn("Gateway has no tags configured, skipping",
				zap.String("gateway", gw.Address))
			continue
		}
		metrics := &GatewayMetrics{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.processPiWebAPIGateway(ctx,gw, metrics, modePiWebAPI)
		}()
	}
	// ── Block until ctx is cancelled (graceful shutdown signal) ──────────────
	<-ctx.Done()
	p.Logger.Info("Shutdown signal received — waiting for gateway goroutines to finish")
	wg.Wait()
	p.Logger.Info("All gateway goroutines stopped cleanly")
}

