package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/helixdevelopment/health-service/internal/model"
)

// DefaultTimeout is the default HTTP timeout for health checks.
const DefaultTimeout = 5 * time.Second

// HealthChecker performs health checks on configured services.
type HealthChecker struct {
	client    *http.Client
	endpoints map[string]string
	timeout   time.Duration
}

// New creates a new HealthChecker with the given service endpoints.
func New(endpoints map[string]string, timeout time.Duration) *HealthChecker {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &HealthChecker{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoints: endpoints,
		timeout:   timeout,
	}
}

// CheckService performs an HTTP GET health check on a single service.
func (c *HealthChecker) CheckService(name, url string) (*model.ServiceHealth, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &model.ServiceHealth{
			Name:           name,
			Status:         model.StatusUnhealthy,
			LastCheckAt:    time.Now().UTC(),
			ResponseTimeMs: 0,
			ErrorMessage:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return &model.ServiceHealth{
			Name:           name,
			Status:         model.StatusUnhealthy,
			LastCheckAt:    time.Now().UTC(),
			ResponseTimeMs: elapsed.Milliseconds(),
			ErrorMessage:   fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Drain body to allow connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	status := model.StatusHealthy
	if resp.StatusCode >= 500 {
		status = model.StatusUnhealthy
	} else if resp.StatusCode >= 400 {
		status = model.StatusDegraded
	}

	return &model.ServiceHealth{
		Name:           name,
		Status:         status,
		LastCheckAt:    time.Now().UTC(),
		ResponseTimeMs: elapsed.Milliseconds(),
	}, nil
}

// CheckAll checks all configured services concurrently and returns aggregated results.
func (c *HealthChecker) CheckAll() (*model.SystemHealth, error) {
	var wg sync.WaitGroup
	results := make([]model.ServiceHealth, 0, len(c.endpoints))
	var mu sync.Mutex

	for name, url := range c.endpoints {
		wg.Add(1)
		go func(n, u string) {
			defer wg.Done()
			sh, _ := c.CheckService(n, u)
			mu.Lock()
			results = append(results, *sh)
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()

	overall := model.StatusHealthy
	for _, svc := range results {
		if svc.Status == model.StatusUnhealthy {
			overall = model.StatusUnhealthy
			break
		}
		if svc.Status == model.StatusDegraded && overall == model.StatusHealthy {
			overall = model.StatusDegraded
		}
	}

	return &model.SystemHealth{
		OverallStatus: overall,
		Services:      results,
		CheckedAt:     time.Now().UTC(),
	}, nil
}

// CheckServices checks a subset of services by name.
func (c *HealthChecker) CheckServices(names []string) (*model.SystemHealth, error) {
	var wg sync.WaitGroup
	results := make([]model.ServiceHealth, 0, len(names))
	var mu sync.Mutex

	for _, name := range names {
		url, ok := c.endpoints[name]
		if !ok {
			mu.Lock()
			results = append(results, model.ServiceHealth{
				Name:           name,
				Status:         model.StatusUnhealthy,
				LastCheckAt:    time.Now().UTC(),
				ResponseTimeMs: 0,
				ErrorMessage:   "service not configured",
			})
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(n, u string) {
			defer wg.Done()
			sh, _ := c.CheckService(n, u)
			mu.Lock()
			results = append(results, *sh)
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()

	overall := model.StatusHealthy
	for _, svc := range results {
		if svc.Status == model.StatusUnhealthy {
			overall = model.StatusUnhealthy
			break
		}
		if svc.Status == model.StatusDegraded && overall == model.StatusHealthy {
			overall = model.StatusDegraded
		}
	}

	return &model.SystemHealth{
		OverallStatus: overall,
		Services:      results,
		CheckedAt:     time.Now().UTC(),
	}, nil
}
