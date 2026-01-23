# UptimeMonitor Component Design

**Traceability:** ARCH-014

## 1. Data Structures & Types

```go
package monitoring

import (
    "context"
    "time"
)

type ServiceEndpoint struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    URL         string            `json:"url"`
    Method      string            `json:"method"`
    Headers     map[string]string `json:"headers"`
    Interval    time.Duration     `json:"interval"`
    Timeout     time.Duration     `json:"timeout"`
    IsCritical  bool              `json:"is_critical"`
    LastCheck   time.Time         `json:"last_check"`
    NextCheck   time.Time         `json:"next_check"`
}

type HealthCheckResult struct {
    EndpointID   string        `json:"endpoint_id"`
    Timestamp    time.Time     `json:"timestamp"`
    StatusCode   int           `json:"status_code"`
    ResponseTime time.Duration `json:"response_time"`
    Success      bool          `json:"success"`
    ErrorMessage string        `json:"error_message,omitempty"`
}

type UptimeStatus struct {
    EndpointID        string         `json:"endpoint_id"`
    TotalChecks       int64          `json:"total_checks"`
    SuccessfulChecks  int64          `json:"successful_checks"`
    FailedChecks      int64          `json:"failed_checks"`
    UptimePercentage  float64        `json:"uptime_percentage"`
    LastSuccess       time.Time      `json:"last_success,omitempty"`
    LastFailure       time.Time      `json:"last_failure,omitempty"`
    CurrentStreak     time.Duration  `json:"current_streak"`
    LastDowntime      time.Time      `json:"last_downtime,omitempty"`
    RecoveryTime      time.Time      `json:"recovery_time,omitempty"`
}

type AlertConfig struct {
    ID              string        `json:"id"`
    EndpointID      string        `json:"endpoint_id"`
    AlertType       AlertType     `json:"alert_type"`
    Threshold       int           `json:"threshold"`
    TimeWindow      time.Duration `json:"time_window"`
    Cooldown        time.Duration `json:"cooldown"`
    NotificationURL string        `json:"notification_url"`
    IsEnabled       bool          `json:"is_enabled"`
}

type AlertType string

const (
    AlertTypeConsecutiveFailures AlertType = "consecutive_failures"
    AlertTypeErrorRate           AlertType = "error_rate"
    AlertTypeResponseTime        AlertType = "response_time"
    AlertTypeDowntime            AlertType = "downtime"
)

type Alert struct {
    ID          string     `json:"id"`
    EndpointID  string     `json:"endpoint_id"`
    AlertType   AlertType  `json:"alert_type"`
    Severity    string     `json:"severity"`
    Message     string     `json:"message"`
    CreatedAt   time.Time  `json:"created_at"`
    ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
    IsActive    bool       `json:"is_active"`
}

type MonitorConfig struct {
    PollInterval   time.Duration    `json:"poll_interval"`
    Timeout        time.Duration    `json:"timeout"`
    Retries        int              `json:"retries"`
    RetryDelay     time.Duration    `json:"retry_delay"`
    GCPMetricPrefix string          `json:"gcp_metric_prefix"`
    EnableAlerts   bool             `json:"enable_alerts"`
    AlertConfig    AlertConfig      `json:"alert_config"`
}

type UptimeRepository interface {
    SaveHealthCheck(ctx context.Context, result HealthCheckResult) error
    GetUptimeStatus(ctx context.Context, endpointID string, timeWindow time.Duration) (*UptimeStatus, error)
    GetHealthCheckHistory(ctx context.Context, endpointID string, limit int) ([]HealthCheckResult, error)
    SaveAlert(ctx context.Context, alert Alert) error
    GetActiveAlerts(ctx context.Context) ([]Alert, error)
    ResolveAlert(ctx context.Context, alertID string) error
}

type GCPMonitoringClient interface {
    WriteMetric(ctx context.Context, metricName string, value float64, labels map[string]string) error
    CreateUptimeCheck(ctx context.Context, config map[string]interface{}) (string, error)
    GetUptimeCheckResults(ctx context.Context, checkID string, timeWindow time.Duration) ([]map[string]interface{}, error)
}

type UptimeMonitor struct {
    config         MonitorConfig
    endpoints      map[string]*ServiceEndpoint
    repository     UptimeRepository
    gcpClient      GCPMonitoringClient
    scheduler      *Scheduler
    alertManager   *AlertManager
   mu             sync.RWMutex
    isRunning      bool
    cancel         context.CancelFunc
}

type Scheduler struct {
    endpoints map[string]*ServiceEndpoint
    interval  time.Duration
    tick      chan struct{}
    stop      chan struct{}
}

type AlertManager struct {
    config     AlertConfig
    repository UptimeRepository
    gcpClient  GCPMonitoringClient
    mu         sync.RWMutex
    activeAlerts map[string]Alert
}

type CheckResult struct {
    endpoint    *ServiceEndpoint
    result      HealthCheckResult
    needsAlert  bool
    alertType   AlertType
}
```

## 2. Logic & Algorithms

### 2.1 Main Monitoring Loop

```
1. START_MONITORING
2. Initialize context with cancellation
3. Load all service endpoints from configuration
4. Start scheduler goroutine
5. FOR each endpoint in endpoints:
6.     Schedule first health check at current time + jitter
7. END FOR
8. Start worker pool for executing health checks
9. Start result processor goroutine
10. Start alert manager goroutine
11. WAIT until context cancellation
12. On cancellation: graceful shutdown with timeout
13. END_WAIT
14. STOP_MONITORING
```

### 2.2 Health Check Execution

```
1. EXECUTE_HEALTH_CHECK(endpoint)
2. Create HTTP client with timeout from endpoint config
3. Build request with endpoint URL, method, and headers
4. Add authentication headers if configured
5. record start_time
6. TRY:
7.     Execute HTTP request
8.     IF error occurs:
9.         Record failure with error message
10.        RETURN CheckResult with success=false
11.    END IF
12.    IF status_code >= 200 AND status_code < 300:
13.        Record success with response time
14.        RETURN CheckResult with success=true
15.    ELSE:
16.        Record failure with status code
17.        RETURN CheckResult with success=false
18.    END IF
19. CATCH timeout:
20.    Record failure with timeout error
21. RETURN CheckResult with success=false
22. FINALLY:
23.     Calculate response_time = now - start_time
24.     Update endpoint.last_check
25.     Schedule next check
```

### 2.3 Scheduler Logic

```
1. SCHEDULE_CHECK(endpoint)
2. Calculate next check time: endpoint.next_check
3. Add to priority queue sorted by next_check time
4. IF queue was empty before add:
5.     Signal scheduler to wake up
6. END IF

7. SCHEDULER_LOOP
8. FOR:
9.     peek at next scheduled check
10.    sleep until check_time or stop signal
11.    IF stop signal received:
12.        BREAK
13.    END IF
14.    pop check from queue
15.    send check to worker pool
16.    Schedule next check for this endpoint
```

### 2.4 Uptime Calculation

```
1. CALCULATE_UPTIME(endpoint_id, time_window)
2. Fetch health check results from repository
3. Filter results within time_window
4. total_checks = count(results)
5. IF total_checks == 0:
6.     RETURN UptimeStatus with uptime_percentage=100
7. END IF
8. successful_checks = count(results where success=true)
9. failed_checks = count(results where success=false)
10. uptime_percentage = (successful_checks / total_checks) * 100
11. Calculate current_streak from last successful check
12. RETURN UptimeStatus with all computed values
```

### 2.5 Alert Evaluation

```
1. EVALUATE_ALERTS(endpoint_id, result)
2. Fetch recent check results within alert_config.time_window
3. FOR each enabled alert rule:
4.     SWITCH alert_type:
5.     CASE consecutive_failures:
6.         count = count consecutive failures from latest
7.         IF count >= threshold:
8.             Trigger alert
9.         END IF
10.    CASE error_rate:
11.        error_count = count failed checks
12.        error_rate = error_count / total_checks
13.        IF error_rate >= threshold/100:
14.            Trigger alert
15.        END IF
16.    CASE response_time:
17.        avg_response = average response times
18.        IF avg_response >= threshold:
19.            Trigger alert
20.        END IF
21.    CASE downtime:
22.        IF result.success == false AND is_critical:
23.            Trigger immediate alert
24.        END IF
25. END SWITCH
26. END FOR
27. IF no conditions met AND active alert exists:
28.     Resolve alert
```

## 3. State Management & Error Handling

### 3.1 Possible Error States

| Error State | Cause | Transition |
|-------------|-------|------------|
| HealthCheckTimeout | Endpoint unresponsive within timeout period | Automatic retry after retry_delay |
| ConnectionRefused | Service not accepting connections | Retry with exponential backoff |
| DNSResolutionFailure | Endpoint URL invalid or DNS down | Log error, disable endpoint |
| TLSHandshakeFailure | Certificate expired or invalid | Log error, notify via alert |
| HTTPError | Non-2xx status code received | Evaluate against alert thresholds |
| RepositoryWriteFailed | Database connection issue | Queue for retry, escalate if persists |
| GCPWriteFailed | Cloud Monitoring API unavailable | Local buffer, retry on recovery |
| SchedulerStopped | Shutdown signal received | Graceful stop with pending checks completion |
| WorkerPoolExhausted | All workers busy | Expand pool or queue with overflow protection |

### 3.2 State Transitions

```
IDLE -> RUNNING:
    Trigger: Start() called
    Action: Initialize workers, start scheduler

RUNNING -> PAUSED:
    Trigger: Pause() called
    Action: Stop scheduler, drain pending checks, keep results

PAUSED -> RUNNING:
    Trigger: Resume() called
    Action: Restart scheduler with pending checks

RUNNING -> SHUTTING_DOWN:
    Trigger: Stop() called
    Action: Signal workers to complete, reject new checks

SHUTTING_DOWN -> STOPPED:
    Trigger: All workers completed or timeout
    Action: Close all connections, cleanup resources

FAILED -> STOPPED:
    Trigger: Unrecoverable error
    Action: Log error, cleanup, notify administrators
```

### 3.3 Error Recovery Strategies

```
1. HEALTH_CHECK_TIMEOUT:
   - Retry up to config.retries times
   - Each retry doubles the timeout
   - After max retries, mark as failed
   - Trigger consecutive_failure alert if threshold reached

2. REPOSITORY_WRITE_FAILED:
   - Retry with exponential backoff (100ms, 200ms, 400ms...)
   - Max 5 retries before buffered write
   - Buffered writes have max size of 1000 entries
   - Flush buffer on next successful write

3. GCP_WRITE_FAILED:
   - Same retry strategy as repository
   - On persistent failure, log to local file
   - Replay local logs on recovery

4. WORKER_PANIC:
   - Recover with defer recover()
   - Log panic with stack trace
   - Restart worker goroutine
   - Re-queue the check that caused panic
```

## 4. Component Interfaces

### 4.1 Public Interface

```go
type UptimeMonitor interface {
    // Lifecycle management
    Start(ctx context.Context) error
    Pause() error
    Resume() error
    Stop() error

    // Endpoint management
    AddEndpoint(ctx context.Context, endpoint ServiceEndpoint) error
    RemoveEndpoint(ctx context.Context, endpointID string) error
    UpdateEndpoint(ctx context.Context, endpoint ServiceEndpoint) error
    GetEndpoints(ctx context.Context) ([]ServiceEndpoint, error)

    // Monitoring operations
    ForceCheck(ctx context.Context, endpointID string) error
    GetUptimeStatus(ctx context.Context, endpointID string, window time.Duration) (*UptimeStatus, error)
    GetHealthCheckHistory(ctx context.Context, endpointID string, limit int) ([]HealthCheckResult, error)

    // Alert management
    GetActiveAlerts(ctx context.Context) ([]Alert, error)
    AcknowledgeAlert(ctx context.Context, alertID string) error
    ConfigureAlert(ctx context.Context, config AlertConfig) error
}
```

### 4.2 Internal Function Signatures

```go
func (m *UptimeMonitor) executeHealthCheck(ctx context.Context, endpoint *ServiceEndpoint) HealthCheckResult

func (m *UptimeMonitor) processResult(ctx context.Context, result HealthCheckResult)

func (m *UptimeMonitor) calculateUptime(ctx context.Context, endpointID string, window time.Duration) (*UptimeStatus, error)

func (m *UptimeMonitor) evaluateAlerts(ctx context.Context, endpointID string, result HealthCheckResult)

func (m *UptimeMonitor) triggerAlert(ctx context.Context, alert Alert) error

func (m *UptimeMonitor) resolveAlert(ctx context.Context, alertID string) error

func (m *Scheduler) scheduleCheck(endpoint *ServiceEndpoint)

func (m *Scheduler) run(ctx context.Context)

func (m *AlertManager) evaluateRules(ctx context.Context, endpointID string, result HealthCheckResult) []AlertType

func (m *AlertManager) trigger(ctx context.Context, alert Alert) error

func (m *AlertManager) resolve(ctx context.Context, alertID string) error

func writeMetricToGCP(ctx context.Context, client GCPMonitoringClient, metricName string, value float64, labels map[string]string) error
```

### 4.3 Configuration

```yaml
monitoring:
  uptime:
    poll_interval: 30s
    timeout: 10s
    retries: 3
    retry_delay: 5s
    gcp_metric_prefix: "custom.googleapis.com/mealswapp/uptime"
    enable_alerts: true
    worker_pool_size: 10
    max_queue_size: 1000

  endpoints:
    - id: "api-gateway"
      name: "API Gateway"
      url: "https://api.mealswapp.com/health"
      method: "GET"
      interval: 30s
      timeout: 5s
      is_critical: true

    - id: "database-primary"
      name: "Primary Database"
      url: "tcp://db.mealswapp.com:5432"
      method: "TCP"
      interval: 15s
      timeout: 3s
      is_critical: true

    - id: "redis-cache"
      name: "Redis Cache"
      url: "redis://cache.mealswapp.com:6379"
      method: "PING"
      interval: 15s
      timeout: 2s
      is_critical: false

  alerts:
    - endpoint_id: "api-gateway"
      alert_type: "consecutive_failures"
      threshold: 3
      time_window: 60s
      cooldown: 300s
      severity: "critical"
      is_enabled: true

    - endpoint_id: "*"
      alert_type: "error_rate"
      threshold: 10
      time_window: 300s
      cooldown: 600s
      severity: "warning"
      is_enabled: true
```

### 4.4 GCP Cloud Monitoring Integration

```go
type GCPMetricsExporter struct {
    projectID    string
    metricClient *monitoring.MetricClient
    buffer       []monitoring.TimeSeries
    bufferMu     sync.Mutex
    flushInterval time.Duration
    maxBufferSize int
}

func (e *GCPMetricsExporter) WriteMetric(ctx context.Context, metricName string, value float64, labels map[string]string) error {
    timeseries := &monitoring.TimeSeries{
        Metric: &monitoring.Metric{
            Type: fmt.Sprintf("%s/%s", e.metricPrefix, metricName),
            Labels: labels,
        },
        Points: []*monitoring.Point{
            {
                Interval: &monitoring.TimeInterval{
                    EndTime: timestamppb.Now(),
                },
                Value: &monitoring.TypedValue{
                    Value: &monitoring.TypedValue_DoubleValue{DoubleValue: value},
                },
            },
        },
    }

    e.bufferMu.Lock()
    e.buffer = append(e.buffer, *timeseries)
    shouldFlush := len(e.buffer) >= e.maxBufferSize
    e.bufferMu.Unlock()

    if shouldFlush {
        return e.Flush(ctx)
    }
    return nil
}

func (e *GCPMetricsExporter) Flush(ctx context.Context) error {
    e.bufferMu.Lock()
    if len(e.buffer) == 0 {
        e.bufferMu.Unlock()
        return nil
    }
    toFlush := e.buffer
    e.buffer = nil
    e.bufferMu.Unlock()

    req := &monitoring.CreateTimeSeriesRequest{
        Name: fmt.Sprintf("projects/%s", e.projectID),
        TimeSeries: toFlush,
    }

    return e.metricClient.CreateTimeSeries(ctx, req)
}
```

### 4.5 HTTP Health Check Implementation

```go
func executeHTTPHealthCheck(ctx context.Context, endpoint *ServiceEndpoint, timeout time.Duration) HealthCheckResult {
    client := &http.Client{
        Timeout: timeout,
        Transport: &http.Transport{
            MaxIdleConns:        10,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }

    req, err := http.NewRequestWithContext(ctx, endpoint.Method, endpoint.URL, nil)
    if err != nil {
        return HealthCheckResult{
            EndpointID:   endpoint.ID,
            Timestamp:    time.Now(),
            Success:      false,
            ErrorMessage: fmt.Sprintf("failed to create request: %v", err),
        }
    }

    for key, value := range endpoint.Headers {
        req.Header.Set(key, value)
    }

    if authToken := getAuthToken(ctx); authToken != "" {
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
    }

    startTime := time.Now()
    resp, err := client.Do(req)
    responseTime := time.Since(startTime)

    if err != nil {
        return HealthCheckResult{
            EndpointID:    endpoint.ID,
            Timestamp:     startTime,
            ResponseTime:  responseTime,
            Success:       false,
            ErrorMessage:  err.Error(),
        }
    }
    defer resp.Body.Close()

    return HealthCheckResult{
        EndpointID:   endpoint.ID,
        Timestamp:    startTime,
        StatusCode:   resp.StatusCode,
        ResponseTime: responseTime,
        Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
    }
}
```

### 4.6 TCP Health Check Implementation

```go
func executeTCPHealthCheck(ctx context.Context, endpoint *ServiceEndpoint, timeout time.Duration) HealthCheckResult {
    startTime := time.Now()

    dialer := &net.Dialer{
        Timeout: timeout,
    }

    conn, err := dialer.DialContext(ctx, "tcp", endpoint.URL)
    if err != nil {
        return HealthCheckResult{
            EndpointID:    endpoint.ID,
            Timestamp:     startTime,
            ResponseTime:  time.Since(startTime),
            Success:       false,
            ErrorMessage:  fmt.Sprintf("TCP connection failed: %v", err),
        }
    }
    defer conn.Close()

    return HealthCheckResult{
        EndpointID:    endpoint.ID,
        Timestamp:     startTime,
        ResponseTime:  time.Since(startTime),
        Success:       true,
    }
}
```

### 4.7 Redis PING Health Check Implementation

```go
func executeRedisHealthCheck(ctx context.Context, endpoint *ServiceEndpoint, timeout time.Duration) HealthCheckResult {
    startTime := time.Now()

    rdb := redis.NewClient(&redis.Options{
        Addr:     endpoint.URL,
        DialTimeout:  timeout,
        ReadTimeout:  timeout,
    })
    defer rdb.Close()

    err := rdb.Ping(ctx).Err()
    if err != nil {
        return HealthCheckResult{
            EndpointID:    endpoint.ID,
            Timestamp:     startTime,
            ResponseTime:  time.Since(startTime),
            Success:       false,
            ErrorMessage:  fmt.Sprintf("Redis PING failed: %v", err),
        }
    }

    return HealthCheckResult{
        EndpointID:    endpoint.ID,
        Timestamp:     startTime,
        ResponseTime:  time.Since(startTime),
        Success:       true,
    }
}
```
