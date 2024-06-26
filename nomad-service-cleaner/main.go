package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/pirogoeth/apps/pkg/system"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	metricDanglingServicesFound = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dangling_services_found_total",
		Help: "Total number of dangling services found",
	})
	metricDanglingServicesCleaned = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dangling_services_cleaned_total",
		Help: "Total number of dangling services cleaned",
	})
	metricDanglingServicesProcessTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "dangling_services_process_time",
		Help: "Amount of time taken to search & process dangling services",
	})
)

type danglingSvc struct {
	namespace   string
	serviceName string
	serviceID   string
}

type App struct {
	cfg *Config
}

func main() {
	logging.Setup(logging.WithAppName("nomad-service-cleaner"))

	prometheus.MustRegister(
		metricDanglingServicesCleaned,
		metricDanglingServicesFound,
		metricDanglingServicesProcessTime,
	)

	cfg, err := config.Load[Config]()
	if err != nil {
		logrus.Fatalf("could not start (config): %s", err)
	}

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Nomad client
	nomadOpts := nomadApi.DefaultConfig()
	nomadClient, err := nomadApi.NewClient(nomadOpts)
	if err != nil {
		logrus.Fatalf("could not create nomad client: %w", err)
	}
	if err := system.NomadClientHealthCheck(ctx, nomadClient); err != nil {
		panic(fmt.Errorf("nomad client not healthy: %w", err))
	}
	system.RegisterNomadClientReadiness(nomadClient)
	defer nomadClient.Close()

	sched, err := gocron.NewScheduler()
	if err != nil {
		logrus.Fatalf("could not created scheduler: %w", err)
	}

	_, err = sched.NewJob(gocron.DurationJob(5*time.Minute), gocron.NewTask(func() {
		logrus.Debugf("Running dangling service cleaner")
		t := prometheus.NewTimer(metricDanglingServicesProcessTime)
		if err := app.cleanDanglingServices(ctx, nomadClient); err != nil {
			logrus.Fatalf("could not clean dangling services: %w", err)
		}
		t.ObserveDuration()
	}))
	if err != nil {
		panic(fmt.Errorf("could not schedule job: %w", err))
	}

	sched.Start()

	router := system.DefaultRouter()
	router.Run(app.cfg.HTTP.ListenAddress)

	sched.Shutdown()
	cancel()
}

func (app *App) cleanDanglingServices(ctx context.Context, nomadClient *nomadApi.Client) error {
	// Collect all namespaces
	namespaces, _, err := nomadClient.Namespaces().List(nil)
	if err != nil {
		return fmt.Errorf("could not list namespaces: %w", err)
	}

	// Collect service names in each namespace
	allNamespaceSvcs := make([]*nomadApi.ServiceRegistrationListStub, 0)
	for _, namespace := range namespaces {
		services, _, err := nomadClient.Services().List(&nomadApi.QueryOptions{
			Namespace: namespace.Name,
		})
		if err != nil {
			return fmt.Errorf("could not list services in namespace %s: %w", namespace.Name, err)
		}

		allNamespaceSvcs = append(allNamespaceSvcs, services...)
	}

	danglingSvcs := make([]*danglingSvc, 0)
	// Collect attached allocations for each service
	for _, namespaceSvcs := range allNamespaceSvcs {
		namespace := namespaceSvcs.Namespace
		for _, serviceEntry := range namespaceSvcs.Services {
			serviceName := serviceEntry.ServiceName
			svcDetails, _, err := nomadClient.Services().Get(serviceName, &nomadApi.QueryOptions{
				Namespace: namespace,
			})
			if err != nil {
				return fmt.Errorf("could not get service %s: %w", serviceName, err)
			}
			for _, svcRegistration := range svcDetails {
				alloc, _, err := nomadClient.Allocations().Info(svcRegistration.AllocID, &nomadApi.QueryOptions{
					Namespace: namespace,
				})
				if err != nil && strings.Contains(err.Error(), "alloc not found") {
					// I don't know if this is actually Nomad's 404 return, check this ^^
					// Add this service ID and allocation ID to list of dangling entries
					danglingSvcs = append(danglingSvcs, &danglingSvc{
						namespace:   namespace,
						serviceName: serviceName,
						serviceID:   svcRegistration.ID,
					})
				} else if err != nil {
					return fmt.Errorf("could not get allocation %s / %s in ns %s: %w (%#v)",
						svcRegistration.AllocID,
						serviceName,
						namespace,
						err,
						err)
				} else if alloc.ClientStatus == nomadApi.AllocClientStatusFailed {
					// TODO: Check if alloc has been in terminal status > 5min or desired state is terminal??
					// Add this service ID and allocation ID to list of dangling entries
					danglingSvcs = append(danglingSvcs, &danglingSvc{
						namespace:   namespace,
						serviceName: serviceName,
						serviceID:   svcRegistration.ID,
					})
				}
			}
		}
	}

	if len(danglingSvcs) > 0 {
		metricDanglingServicesFound.Add(float64(len(danglingSvcs)))

		for _, danglingSvc := range danglingSvcs {
			logrus.Infof("Found dangling service: %s/%s\n", danglingSvc.namespace, danglingSvc.serviceID)
			// Delete the service
			if _, err := nomadClient.Services().Delete(danglingSvc.serviceName, danglingSvc.serviceID, &nomadApi.WriteOptions{
				Namespace: danglingSvc.namespace,
			}); err != nil {
				return fmt.Errorf("could not deregister service %s: %w", danglingSvc.serviceID, err)
			}

			metricDanglingServicesCleaned.Inc()
		}
	}

	return nil
}
