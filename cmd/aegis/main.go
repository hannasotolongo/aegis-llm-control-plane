package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hannasotolongo/aegis-llm-control-plane/internal/api"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/cluster"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/predictor"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/risk"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/scheduler"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/simulator"
	"github.com/hannasotolongo/aegis-llm-control-plane/internal/telemetry"
)

func main() {
	store := cluster.NewInMemoryStateStore()

	sim := simulator.New(store)

	if err := sim.SeedWorkers(
		context.Background(),
		4,
	); err != nil {
		log.Fatalf(
			"seed simulator workers: %v",
			err,
		)
	}

	if err := sim.SeedWorkloads(
		context.Background(),
	); err != nil {
		log.Fatalf(
			"seed simulator workloads: %v",
			err,
		)
	}

	telemetryStore := telemetry.NewInMemoryStore()

	collector := telemetry.NewCollector(
		telemetryStore,
	)

	telemetryService := telemetry.NewService(
		store,
		collector,
		5*time.Second,
	)

	trendPredictor := predictor.NewTrendPredictor()

	predictionResults := predictor.NewResultStore()

	predictionService := predictor.NewService(
		telemetryStore,
		trendPredictor,
		predictionResults,
		5*time.Second,
		30*time.Second,
	)

	riskEvaluator := risk.NewEvaluator()

	riskAwareScheduler := scheduler.NewRiskAwareService(
		store,
		predictionResults,
		riskEvaluator,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		err := sim.Run(
			ctx,
			2*time.Second,
		)

		if err != nil &&
			!errors.Is(err, context.Canceled) {
			log.Printf(
				"simulator stopped: %v",
				err,
			)
		}
	}()

	go func() {
		err := predictionService.Run(ctx)

		if err != nil &&
			!errors.Is(err, context.Canceled) {
			log.Printf(
				"prediction service stopped: %v",
				err,
			)
		}
	}()

	go func() {
		err := telemetryService.Run(ctx)

		if err != nil &&
			!errors.Is(err, context.Canceled) {
			log.Printf(
				"telemetry service stopped: %v",
				err,
			)
		}
	}()

	go func() {
		err := riskAwareScheduler.Run(
			ctx,
			2*time.Second,
		)

		if err != nil &&
			!errors.Is(err, context.Canceled) {
			log.Printf(
				"scheduler stopped: %v",
				err,
			)
		}
	}()

	apiServer := api.NewServer(
		store,
		riskAwareScheduler,
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: apiServer.Handler(),
	}

	go func() {
		log.Printf(
			"Aegis control plane listening on %s",
			server.Addr,
		)

		err := server.ListenAndServe()

		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {
			log.Printf(
				"HTTP server error: %v",
				err,
			)

			stop()
		}
	}()

	<-ctx.Done()

	log.Printf("shutting down Aegis")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(
		shutdownCtx,
	); err != nil {
		log.Printf(
			"HTTP shutdown error: %v",
			err,
		)
	}

	records, err := telemetryStore.List(
		context.Background(),
	)
	if err != nil {
		log.Printf(
			"read telemetry: %v",
			err,
		)
		return
	}

	if err := exportTelemetry(records); err != nil {
		log.Printf(
			"export telemetry: %v",
			err,
		)
		return
	}

	log.Printf(
		"exported %d telemetry records to aegis-telemetry.csv",
		len(records),
	)
}

func exportTelemetry(
	records []telemetry.Record,
) error {
	file, err := os.Create(
		"aegis-telemetry.csv",
	)
	if err != nil {
		return err
	}

	defer file.Close()

	return telemetry.WriteCSV(
		file,
		records,
	)
}
