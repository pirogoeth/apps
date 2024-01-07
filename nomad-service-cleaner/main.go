package main

import (
	"context"
	"fmt"

	nomadApi "github.com/hashicorp/nomad/api"
	"github.com/pirogoeth/apps/pkg/config"
	"github.com/pirogoeth/apps/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
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
	logging.Setup()

	cfg, err := config.Load[Config]()
	if err != nil {
		panic(fmt.Errorf("could not start (config): %w", err))
	}

	app := &App{cfg}

	ctx, cancel := context.WithCancel(context.Background())

	// Create Nomad client
	nomadOpts := nomadApi.DefaultConfig()
	nomadClient, err := nomadApi.NewClient(nomadOpts)
	if err != nil {
		panic(fmt.Errorf("could not create nomad client: %w", err))
	}
	defer nomadClient.Close()

	if err := app.cleanDanglingServices(ctx, nomadClient); err != nil {
		panic(fmt.Errorf("could not clean dangling services: %w", err))
	}

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
				if err != nil && err.Error() == "404 (Not Found)" {
					// I don't know if this is actually Nomad's 404 return, check this ^^
					// Add this service ID and allocation ID to list of dangling entries
					danglingSvcs = append(danglingSvcs, &danglingSvc{
						namespace:   namespace,
						serviceName: serviceName,
						serviceID:   svcRegistration.ID,
					})
				} else if err != nil {
					return fmt.Errorf("could not get allocation %s: %w (%#v)", svcRegistration.AllocID, err, err)
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
			fmt.Printf("Found dangling service: %s/%s\n", danglingSvc.namespace, danglingSvc.serviceID)
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
