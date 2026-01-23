# MetricsCollector

**Traceability:** ARCH-014

## 1. Data Structures & Types

```go
package metrics

import (
	"context"
	"time"
)

type MetricType string

const (
	MetricTypeResponseTime   MetricType = "response_time"
	MetricTypeErrorRate      MetricType = "error_rate"
	MetricTypeConcurrentUsers MetricType = "concurrent_users"
	MetricTypeP95Latency     MetricType = "p95_latency"
)

type MetricValue struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}

type MetricsBucket struct {
	StartTime   time.Time            `json:"start_time"`
	EndTime     time.Time            `json:"end_time"`
	Metrics     map[MetricType]float64 `json:"metrics"`
	Labels      map[string]string    `json:"labels"`
}

type CollectorConfig struct {
	CollectionInterval  time.Duration `json:"collection_interval"`
	ReportingInterval   time.Duration `json:"reporting_interval"`
	ProjectID           string        `json:"project_id"`
	MetricsPrefix       string        `json:"metrics_prefix"`
	EnabledMetrics      []MetricType  `json:"enabled_metrics"`
}

type ResponseMetrics struct {
	Endpoint     string        `json:"endpoint"`
	Method       string        `json:"method"`
	StatusCode   int           `json:"status_code"`
	Duration     time.Duration `json:"duration"`
	Timestamp    time.Time     `json:"timestamp"`
	UserID       string        `json:"user_id,omitempty"`
}

type ConcurrentUserTracker struct {
	ActiveUsers map[string]time.Time
	mu          sync.RWMutex
}

type LatencyBucket struct {
	Durations []time.Duration
	mu         sync.Mutex
}

type MetricsCollector interface {
	Start(ctx context.Context) error
	Stop() error
	RecordResponse(ctx context.Context, metrics ResponseMetrics) error
	RecordError(ctx context.Context, endpoint string, err error) error
	IncrementConcurrentUsers(userID string) error
	DecrementConcurrentUsers(userID string) error
	GetMetrics(ctx context.Context, metricType MetricType, duration time.Duration) ([]MetricValue, error)
}

type gcpMetricsCollector struct {
	config        CollectorConfig
	client        *monitoring.MetricClient
	buckets       map[string]*LatencyBucket
	userTracker   *ConcurrentUserTracker
	metricBuffers map[MetricType][]float64
	stopCh        chan struct{}
	wg            sync.WaitGroup
}
```

## 2. Logic & Algorithms

### 2.1 Response Time Collection Flow

```
1. Middleware intercepts incoming HTTP request
   └─ Record request start timestamp
   
2. Middleware intercepts response after handler completes
   └─ Calculate response duration (end - start)
   └─ Extract endpoint, method, status code, user ID
   
3. RecordResponse() called with ResponseMetrics
   └─ Validate metrics data is not nil
   └─ Lock latency bucket for endpoint
   └─ Append duration to bucket's durations slice
   └─ Unlock bucket
   
4. If status code >= 400, record error metrics
   └─ Increment error counter for endpoint
   └─ Log error type and message
   
5. Every collection interval (default: 10s)
   └─ For each endpoint bucket:
      ├─ Calculate P50, P95, P99 percentiles
      ├─ Calculate error rate (errors / total requests)
      ├─ Aggregate into MetricsBucket
      └─ Send to GCP Cloud Monitoring
   
6. Every reporting interval (default: 60s)
   └─ Flush all buffered metrics
   └─ Reset bucket counters
   └─ Emit heartbeat metric
```

### 2.2 Concurrent Users Tracking Flow

```
1. User authentication succeeds
   └─ Middleware extracts user ID from session/token
   └─ Call IncrementConcurrentUsers(userID)
   
2. IncrementConcurrentUsers(userID)
   └─ Lock user tracker mutex
   └─ Add/update userID with current timestamp
   └─ Unlock mutex
   
3. Every 30 seconds, cleanup stale users
   └─ Lock mutex
   └─ For each userID in ActiveUsers:
      ├─ If last activity > 5 minutes ago
      └─ Remove from ActiveUsers
   └─ Unlock mutex
   
4. Record concurrent users count to GCP
   └─ Count entries in ActiveUsers
   └─ Emit metric with value = count
```

### 2.3 P95 Latency Calculation Algorithm

```
function calculateP95Latency(durations []time.Duration) float64:
    if len(durations) == 0:
        return 0.0
    
    sort durations in ascending order
    
    index = int(0.95 * float64(len(durations)))
    if index >= len(durations):
        index = len(durations) - 1
    
    return duration_milliseconds(durations[index])
```

### 2.4 Error Rate Calculation Algorithm

```
function calculateErrorRate(errors int, total int) float64:
    if total == 0:
        return 0.0
    
    return (float64(errors) / float64(total)) * 100.0
```

### 2.5 Metrics Reporting to GCP Cloud Monitoring

```
function reportMetrics(bucket MetricsBucket):
    for metricType, value := range bucket.Metrics:
        metricName = config.MetricsPrefix + "_" + string(metricType)
        
        metric := &monitoring.Point{
            TimeSeries: &monitoring.TimeSeries{
                Metric: &monitoring.Metric{
                    Type: "custom.googleapis.com/" + metricName,
                    Labels: bucket.Labels,
                },
                Points: []*monitoring.Point{
                    {
                        Interval: &monitoring.TimeInterval{
                            StartTime: bucket.StartTime,
                            EndTime:   bucket.EndTime,
                        },
                        Value: &monitoring.TypedValue{
                            DoubleValue: &value,
                        },
                    },
                },
            },
        }
        
        client.CreateTimeSeries(ctx, metric)
```

## 3. State Management & Error Handling

### 3.1 Error States

| Error State | Cause | Handling Strategy |
|-------------|-------|-------------------|
| GCP Connection Timeout | Network issues, GCP unavailable | Retry with exponential backoff (max 3 attempts), log error locally |
| Metric Buffer Full | High throughput, slow GCP ingestion | Drop oldest metrics, log warning, emit overflow counter |
| Invalid Metric Labels | Malformed endpoint names, missing labels | Sanitize labels, replace invalid chars with underscore |
| Context Cancellation | Shutdown signal received | Flush remaining metrics gracefully, return immediately |
| Stale User Data | Concurrent user tracking data corruption | Reset tracker, emit gauge with current count from database |
| Bucket Not Found | Endpoint metrics bucket not initialized | Create new bucket, log debug message |
| percentile Calculation Error | Empty duration slice | Return 0.0, log debug message |

### 3.2 State Transitions

```
┌─────────────────────────────────────────────────────────────────┐
│                        INITIAL STATE                            │
│                    (collector not started)                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      STARTING STATE                              │
│            (config validation, client initialization)            │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
              Success ✓            Failure ✗
                    │                   │
                    ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                    COLLECTING STATE                              │
│    (receiving metrics, buffering, periodic reporting)            │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Internal States:                                          │   │
│  │  • IDLE - waiting for metrics                             │   │
│  │  • BUFFERING - accumulating metrics in memory             │   │
│  │  • REPORTING - sending to GCP Cloud Monitoring            │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
              Stop() called      Critical error
                    │                   │
                    ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                     STOPPING STATE                               │
│              (flush buffer, close connections)                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      STOPPED STATE                               │
│                   (all resources released)                       │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 Recovery Procedures

**GCP Connection Recovery:**
```
1. Detect connection failure (error on CreateTimeSeries)
2. Increment connection failure counter
3. Enter retry loop:
   for attempt := 1; attempt <= 3; attempt++:
       wait_duration = 2^attempt * time.Second
       sleep(wait_duration)
       try again
4. If all attempts fail:
   - Log error with full context
   - Store metrics in local buffer (max 1000 entries)
   - Schedule retry in next collection cycle
5. On successful reconnection:
   - Flush local buffer to GCP
   - Reset failure counter
   - Emit reconnection event
```

**Buffer Overflow Recovery:**
```
1. Detect buffer size > max_capacity
2. Calculate overflow_count = current_size - max_capacity
3. Drop oldest overflow_count metrics
4. Increment overflow counter metric
5. Emit warning log with dropped count
6. Continue collecting new metrics
```

## 4. Component Interfaces

### 4.1 Public Interface Methods

```go
// NewMetricsCollector creates a new MetricsCollector instance
func NewMetricsCollector(ctx context.Context, config CollectorConfig) (MetricsCollector, error)

// Start begins the metrics collection and reporting loop
func (c *gcpMetricsCollector) Start(ctx context.Context) error

// Stop gracefully shuts down the collector
func (c *gcpMetricsCollector) Stop() error

// RecordResponse records a single HTTP response for metrics
func (c *gcpMetricsCollector) RecordResponse(ctx context.Context, metrics ResponseMetrics) error

// RecordError records an error occurrence
func (c *gcpMetricsCollector) RecordError(ctx context.Context, endpoint string, err error) error

// IncrementConcurrentUsers adds a user to the concurrent count
func (c *gcpMetricsCollector) IncrementConcurrentUsers(userID string) error

// DecrementConcurrentUsers removes a user from the concurrent count
func (c *gcpMetricsCollector) DecrementConcurrentUsers(userID string) error

// GetMetrics retrieves historical metrics for analysis
func (c *gcpMetricsCollector) GetMetrics(ctx context.Context, metricType MetricType, duration time.Duration) ([]MetricValue, error)
```

### 4.2 Fiber Middleware Interface

```go
// MetricsMiddleware creates a Fiber middleware for automatic metrics collection
func MetricsMiddleware(collector MetricsCollector) fiber.Handler

// Usage:
// app.Use(metrics.MetricsCollectorMiddleware(collector))
```

### 4.3 Configuration Structure

```go
// DefaultCollectorConfig returns the recommended default configuration
func DefaultCollectorConfig(projectID string) CollectorConfig {
	return CollectorConfig{
		CollectionInterval:  10 * time.Second,
		ReportingInterval:   60 * time.Second,
		ProjectID:           projectID,
		MetricsPrefix:       "mealswapp",
		EnabledMetrics: []MetricType{
			MetricTypeResponseTime,
			MetricTypeErrorRate,
			MetricTypeConcurrentUsers,
			MetricTypeP95Latency,
		},
	}
}
```

### 4.4 Health Check Interface

```go
// HealthCheck verifies the collector is operational
func (c *gcpMetricsCollector) HealthCheck(ctx context.Context) error

// Returns:
// - nil if collector is healthy
// - ErrCollectorStopped if stopped
// - ErrGCPUnavailable if GCP connection failed
// - ErrBufferOverflow if metrics buffer is full
```
