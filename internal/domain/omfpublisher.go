package domain

import (
	"aradel-pi/config"
	"context"
	"fmt"
	"log"
	"sync"
)

// StartOMFPublisher initializes and coordinates data publishing across all gateways.
func (p *Publisher) StartOMFPublisher(ctx context.Context) {
	fmt.Println("🚀 Starting OMF Service...")

	if p.OMFClient == nil || len(p.OMFClient.Gateways) == 0 {
		log.Println("⚠️ No OMF gateways configured.")
		return
	}

	//fmt.Println("🛠️ Setting up OMF Structure...")
	//omfClient.SetupOMF()

	var wg sync.WaitGroup

	// Launch a dedicated goroutine per Gateway for true concurrency and scalability.
	for _, gatewayInfo := range p.OMFClient.Gateways {
		gw := gatewayInfo
		if len(gw.Tags) == 0 {
			log.Printf("⚠️ Gateway %s has no tags configured, skipping...", gw.Address)
			continue
		}

		wg.Add(1)
		metrics := &GatewayMetrics{}
		go func(g config.Gateway) {
			defer wg.Done()
			p.processOMFGateway(ctx, g, metrics, modeOMF)
		}(gw)
	}

	// ── Block until ctx is cancelled (graceful shutdown signal) ──────────────
	<-ctx.Done()
	log.Println("Shutdown signal received — waiting for gateway goroutines to finish")
	wg.Wait()
	log.Println("🛑 OMF Service Shutdown.")
}
