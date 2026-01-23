## FILE: AlertManager.md
**Traceability:** ARCH-014

### 1. Data Structures & Types

```go
package monitoring

import (
    "time"
    "context"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
    SeverityCritical AlertSeverity = "critical"
    SeverityWarning  AlertSeverity = "warning"
    SeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
    StatusFiring   AlertStatus = "firing"
    StatusResolved AlertStatus = "resolved"
    StatusPending  AlertStatus = "pending"
)

// AlertType represents the category of the alert
type AlertType string

const (
    AlertTypeUptime          AlertType = "uptime"
    AlertTypeLatency         AlertType = "latency"
    AlertTypeErrorRate       AlertType = "error_rate"
    AlertTypeConcurrentUsers AlertType = "concurrent_users"
    AlertTypeBackupFailed    AlertType = "backup_failed"
    AlertTypeBackupRestore   AlertType = "backup_restore_test"
)

// Alert represents a system alert
type Alert struct {
    ID          string
    Name        string
    Description string
    Severity    AlertSeverity
    Status      AlertStatus
    Type        AlertType
    Labels      map[string]string
    Annotations map[string]string
    StartsAt    time.Time
    EndsAt      *time.Time
    Generator   string
}

// AlertRule defines a condition for triggering alerts
type AlertRule struct {
    ID          string
    Name        string
    Expr        string              // PromQL-like expression
    For         time.Duration       // Duration before firing
    Labels      map[string]string
    Annotations map[string]string
    Severity    AlertSeverity
    Type        AlertType
    Enabled     bool
}

// AlertManagerConfig holds configuration for the AlertManager
type AlertManagerConfig struct {
    EvaluationInterval time.Duration
    RetentionPeriod    time.Duration
    NotificationDelay  time.Duration
    GCInterval         time.Duration
    ResendDelay        time.Duration
    ProjectID          string
    Region             string
}

// NotificationChannel represents a destination for alert notifications
type NotificationChannel struct {
    ID      string
    Type    string  // email, webhook, slack, pagerduty
    Config  map[string]interface{}
    Labels  map[string]string
}

// AlertHistory represents a historical record of alert state changes
type AlertHistory struct {
    AlertID    string
    FromStatus AlertStatus
    ToStatus   AlertStatus
    Timestamp  time.Time
    Reason     string
}

// MetricsQuery represents a query to GCP Cloud Monitoring
type MetricsQuery struct {
    MetricType string
    Filter     string
    Aggregator string
    Alignment  time.Duration
    Window     time.Duration
}

// ThresholdConfig defines threshold-based alert conditions
type ThresholdConfig struct {
    MetricName  string
    Operator    string  // gt, lt, gte, lte
    Value       float64
    Duration    time.Duration
    Description string
}
```

### 2. Logic & Algorithms

**Algorithm 2.1: Alert Evaluation Loop**

```
FUNCTION StartAlertEvaluation(ctx context.Context)
    WHILE ctx is not cancelled
        FOR EACH AlertRule IN enabledAlertRules
            result := EvaluateAlertRule(rule)
            IF result.IsFiring AND rule.Type == "threshold"
                HandleThresholdAlert(rule, result)
            ELSE IF result.IsFiring AND rule.Type == "uptime"
                HandleUptimeAlert(rule, result)
            ELSE IF result.IsFiring AND rule.Type == "backup"
                HandleBackupAlert(rule, result)
            END IF
        END FOR
        Sleep(evaluationInterval)
    END WHILE
END FUNCTION

FUNCTION EvaluateAlertRule(rule AlertRule) Result
    metrics := QueryMetrics(rule.MetricQuery)
    value := AggregateMetrics(metrics, rule.Aggregator)
    matches := CompareAgainstThreshold(value, rule.Threshold)
    
    RETURN Result{
        IsFiring: matches,
        Value:    value,
        Timestamp: Now()
    }
END FUNCTION
```

**Algorithm 2.2: Threshold-Based Alert Processing**

```
FUNCTION HandleThresholdAlert(rule AlertRule, result Result)
    existingAlert := FindExistingAlert(rule.ID)
    
    IF existingAlert == nil
        IF result.IsFiring
            newAlert := CreateAlert(rule, result)
            newAlert.Status := StatusPending
            newAlert.StartsAt := Now()
            SaveAlert(newAlert)
            
            IF rule.For == 0 OR rule.For_elapsed(result.StartTime)
                PromoteToFiring(newAlert)
            END IF
        END IF
    ELSE
        IF result.IsFiring
            IF existingAlert.Status == StatusPending
                IF rule.For_elapsed(existingAlert.StartsAt)
                    PromoteToFiring(existingAlert)
                END IF
            END IF
            UpdateAlertEndsAt(existingAlert, nil)
        ELSE
            IF existingAlert.Status == StatusFiring
                ResolveAlert(existingAlert)
            END IF
        END IF
    END IF
END FUNCTION

FUNCTION PromoteToFiring(alert Alert)
    alert.Status := StatusFiring
    alert.Generator := "AlertManager"
    SendNotifications(alert)
    RecordHistory(alert, alert.Status, StatusFiring, "Alert condition met")
END FUNCTION

FUNCTION ResolveAlert(alert Alert)
    now := Now()
    alert.Status := StatusResolved
    alert.EndsAt := &now
    SendNotifications(alert)
    RecordHistory(alert, StatusFiring, StatusResolved, "Alert condition resolved")
END FUNCTION
```

**Algorithm 2.3: Uptime Monitoring Alert**

```
FUNCTION HandleUptimeAlert(rule AlertRule, result Result)
    healthyCount := GetHealthyChecks(rule.ServiceID, rule.Window)
    totalChecks := GetTotalChecks(rule.ServiceID, rule.Window)
    availability := healthyCount / totalChecks
    
    IF availability < rule.AvailabilityThreshold
        alert := FindOrCreateUptimeAlert(rule)
        IF alert.Status != StatusFiring
            PromoteToFiring(alert)
        END IF
        UpdateAlertEndsAt(alert, nil)
    ELSE
        alert := FindExistingAlert(rule.ID)
        IF alert != nil AND alert.Status == StatusFiring
            ResolveAlert(alert)
        END IF
    END IF
END FUNCTION
```

**Algorithm 2.4: Backup Verification Alert**

```
FUNCTION HandleBackupAlert(rule AlertRule, result Result)
    IF result.BackupFailed
        alert := FindOrCreateBackupAlert(rule)
        alert.Severity := SeverityCritical
        PromoteToFiring(alert)
        SendPriorityNotification(alert)
    ELSE IF result.RestoreTestFailed
        alert := FindOrCreateRestoreAlert(rule)
        alert.Severity := SeverityCritical
        PromoteToFiring(alert)
        SendPriorityNotification(alert)
    END IF
END FUNCTION

FUNCTION DailyBackupCheck()
    backupStatus := QueryBackupStatus()
    
    IF backupStatus.LastBackupTime < 24 hours ago
        alert := CreateAlertFromRule("backup_stale")
        alert.Severity := SeverityWarning
        PromoteToFiring(alert)
    END IF
    
    IF backupStatus.RestoreTestStatus == "failed"
        alert := CreateAlertFromRule("backup_restore_failed")
        alert.Severity := SeverityCritical
        PromoteToFiring(alert)
    END IF
END FUNCTION
```

**Algorithm 2.5: Notification Routing**

```
FUNCTION SendNotifications(alert Alert)
    channels := GetChannelsForAlert(alert.Labels)
    
    FOR EACH channel IN channels
        SendToChannel(channel, alert)
    END FOR
END FUNCTION

FUNCTION SendToChannel(channel NotificationChannel, alert Alert)
    payload := FormatAlertPayload(alert)
    
    SWITCH channel.Type
        CASE "email"
            SendEmailNotification(channel, payload)
        CASE "webhook"
            SendWebhookNotification(channel, payload)
        CASE "slack"
            SendSlackNotification(channel, payload)
        CASE "pagerduty"
            TriggerPagerDutyIncident(channel, payload)
    END SWITCH
END FUNCTION
```

**Algorithm 2.6: Alert Garbage Collection**

```
FUNCTION StartGarbageCollection(ctx context.Context)
    WHILE ctx is not cancelled
        resolvedAlerts := FindResolvedAlertsOlderThan(retentionPeriod)
        
        FOR EACH alert IN resolvedAlerts
            RecordFinalState(alert)
            DeleteAlert(alert.ID)
        END FOR
        
        DeleteOldHistory(90 days)
        Sleep(gcInterval)
    END WHILE
END FUNCTION
```

### 3. State Management & Error Handling

**State Transitions:**

| Current State | Event | Next State | Action |
|--------------|-------|------------|--------|
| None | Condition met | Pending | Create alert, start timer |
| Pending | Condition persists | Firing | Send notifications |
| Pending | Condition clears | Resolved | Close alert |
| Firing | Condition clears | Resolved | Send resolved notification |
| Firing | Re-notification timer | Firing | Resend notifications |
| Resolved | None | Archived | GC after retention |

**Error States:**

| Error Condition | Handling Strategy | Recovery Action |
|----------------|-------------------|-----------------|
| GCP Monitoring API timeout | Retry with exponential backoff (max 3 attempts) | Log error, mark as degraded |
| Notification delivery failure | Queue for retry, circuit breaker after 5 failures | Alert on notification failure |
| Alert rule evaluation error | Skip rule, log error, continue with next rule | Log detailed error, alert operator |
| Database connection failure | Buffer alerts in memory, retry connection | FIFO flush when restored |
| Duplicate alert detection | Deduplicate by fingerprint | Merge annotations, update timestamp |

**Alert Fingerprint Calculation:**

```
FUNCTION CalculateFingerprint(alert Alert) string
    components := []string{
        alert.Labels["alertname"],
        alert.Labels["severity"],
        alert.Labels["service"],
        alert.Labels["instance"]
    }
    RETURN SHA256Hash(Join(components, ":"))
END FUNCTION
```

**Deduplication Logic:**

```
FUNCTION DeduplicateAlerts(alerts []Alert) []Alert
    fingerprints := make(map[string]Alert)
    
    FOR EACH alert IN alerts
        fingerprint := CalculateFingerprint(alert)
        IF existing, exists := fingerprints[fingerprint]
            IF alert.StartsAt.Before(existing.StartsAt)
                fingerprints[fingerprint] = alert
            END IF
        ELSE
            fingerprints[fingerprint] = alert
        END IF
    END FOR
    
    RETURN Values(fingerprints)
END FUNCTION
```

**Silence Management:**

```
TYPE Silence struct {
    ID        string
    Matchers  []Matcher
    StartsAt  time.Time
    EndsAt    time.Time
    CreatedBy string
    Comment   string
}

FUNCTION IsAlertSilenced(alert Alert, silences []Silence) bool
    FOR EACH silence IN silences
        IF IsTimeInRange(Now(), silence.StartsAt, silence.EndsAt)
            AND AllMatchersMatch(alert.Labels, silence.Matchers)
            RETURN true
        END IF
    END FOR
    RETURN false
END FUNCTION
```

### 4. Component Interfaces

```go
package monitoring

import (
    "context"
    "time"
)

// AlertManager interface defines the public API for alert management
type AlertManager interface {
    Start(ctx context.Context) error
    Stop() error
    EvaluateAlerts(ctx context.Context) error
    GetActiveAlerts(ctx context.Context) ([]Alert, error)
    GetAlertHistory(ctx context.Context, alertID string, from, to time.Time) ([]AlertHistory, error)
    CreateSilence(ctx context.Context, silence Silence) error
    DeleteSilence(ctx context.Context, id string) error
    SilenceAlert(ctx context.Context, alertID string, duration time.Duration, reason string) error
    GetSilences(ctx context.Context) ([]Silence, error)
    SendTestAlert(ctx context.Context, alert Alert) error
}

// AlertEvaluator interface for evaluating alert conditions
type AlertEvaluator interface {
    EvaluateRule(ctx context.Context, rule AlertRule) (EvaluationResult, error)
    EvaluateExpression(ctx context.Context, expr string, now time.Time) (float64, error)
    QueryTimeSeries(ctx context.Context, query MetricsQuery, start, end time.Time) ([]TimeSeries, error)
}

// NotificationSender interface for sending alert notifications
type NotificationSender interface {
    Send(ctx context.Context, channel NotificationChannel, alert Alert) error
    SendBatch(ctx context.Context, channel NotificationChannel, alerts []Alert) error
    GetChannelStatus(ctx context.Context, channelID string) (ChannelStatus, error)
}

// AlertRepository interface for persisting alerts
type AlertRepository interface {
    SaveAlert(ctx context.Context, alert Alert) error
    UpdateAlert(ctx context.Context, alert Alert) error
    GetAlert(ctx context.Context, id string) (Alert, error)
    GetActiveAlerts(ctx context.Context) ([]Alert, error)
    GetAlertsByRule(ctx context.Context, ruleID string) ([]Alert, error)
    GetAlertsByTimeRange(ctx context.Context, from, to time.Time) ([]Alert, error)
    DeleteAlert(ctx context.Context, id string) error
    SaveAlertHistory(ctx context.Context, history AlertHistory) error
    GetAlertHistory(ctx context.Context, alertID string, from, to time.Time) ([]AlertHistory, error)
}

// UptimeChecker interface for health checks
type UptimeChecker interface {
    CheckEndpoint(ctx context.Context, url string, timeout time.Duration) (UptimeResult, error)
    RunContinuousChecks(ctx context.Context, endpoints []EndpointConfig) error
    GetUptimeStats(ctx context.Context, serviceID string, window time.Duration) (UptimeStats, error)
}

// BackupMonitor interface for backup verification
type BackupMonitor interface {
    CheckLastBackup(ctx context.Context) (BackupStatus, error)
    TestRestore(ctx context.Context, backupID string) (RestoreResult, error)
    RunDailyVerification(ctx context.Context) error
    GetBackupHistory(ctx context.Context, days int) ([]BackupRecord, error)
}

// Implementation: AlertManager

type alertManager struct {
    config        AlertManagerConfig
    evaluator     AlertEvaluator
    notifier      NotificationSender
    repository    AlertRepository
    uptimeChecker UptimeChecker
    backupMonitor BackupMonitor
    ruleCache     []AlertRule
    stopCh        chan struct{}
    wg            sync.WaitGroup
}

func NewAlertManager(
    config AlertManagerConfig,
    evaluator AlertEvaluator,
    notifier NotificationSender,
    repository AlertRepository,
    uptimeChecker UptimeChecker,
    backupMonitor BackupMonitor,
) AlertManager {
    return &alertManager{
        config:        config,
        evaluator:     evaluator,
        notifier:      notifier,
        repository:    repository,
        uptimeChecker: uptimeChecker,
        backupMonitor: backupMonitor,
    }
}

func (am *alertManager) Start(ctx context.Context) error {
    am.stopCh = make(chan struct{})
    
    rules, err := am.loadAlertRules(ctx)
    if err != nil {
        return err
    }
    am.ruleCache = rules
    
    am.wg.Add(1)
    go am.evaluationLoop(ctx)
    
    am.wg.Add(1)
    go am.backupCheckLoop(ctx)
    
    am.wg.Add(1)
    go am.gcLoop(ctx)
    
    return nil
}

func (am *alertManager) Stop() {
    close(am.stopCh)
    am.wg.Wait()
}

func (am *alertManager) EvaluateAlerts(ctx context.Context) error {
    for _, rule := range am.ruleCache {
        if !rule.Enabled {
            continue
        }
        
        result, err := am.evaluator.EvaluateRule(ctx, rule)
        if err != nil {
            log.Printf("Error evaluating rule %s: %v", rule.ID, err)
            continue
        }
        
        am.processEvaluationResult(ctx, rule, result)
    }
    return nil
}

func (am *alertManager) GetActiveAlerts(ctx context.Context) ([]Alert, error) {
    return am.repository.GetActiveAlerts(ctx)
}

func (am *alertManager) GetAlertHistory(ctx context.Context, alertID string, from, to time.Time) ([]AlertHistory, error) {
    return am.repository.GetAlertHistory(ctx, alertID, from, to)
}

func (am *alertManager) CreateSilence(ctx context.Context, silence Silence) error {
    silence.ID = generateID()
    return am.repository.SaveSilence(ctx, silence)
}

func (am *alertManager) DeleteSilence(ctx context.Context, id string) error {
    return am.repository.DeleteSilence(ctx, id)
}

func (am *alertManager) SilenceAlert(ctx context.Context, alertID string, duration time.Duration, reason string) error {
    alert, err := am.repository.GetAlert(ctx, alertID)
    if err != nil {
        return err
    }
    
    silence := Silence{
        ID:        generateID(),
        Matchers:  []Matcher{{Name: "alertname", Value: alert.Name, Operator: "="}},
        StartsAt:  time.Now(),
        EndsAt:    time.Now().Add(duration),
        CreatedBy: "system",
        Comment:   reason,
    }
    
    return am.repository.SaveSilence(ctx, silence)
}

func (am *alertManager) GetSilences(ctx context.Context) ([]Silence, error) {
    return am.repository.GetActiveSilences(ctx)
}

func (am *alertManager) SendTestAlert(ctx context.Context, alert Alert) error {
    alert.ID = generateID()
    alert.Status = StatusFiring
    alert.StartsAt = time.Now()
    
    channels, _ := am.repository.GetNotificationChannels(ctx)
    for _, ch := range channels {
        if err := am.notifier.Send(ctx, ch, alert); err != nil {
            log.Printf("Failed to send test alert to channel %s: %v", ch.ID, err)
        }
    }
    return nil
}

func (am *alertManager) evaluationLoop(ctx context.Context) {
    ticker := time.NewTicker(am.config.EvaluationInterval)
    defer ticker.Stop()
    defer am.wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-am.stopCh:
            return
        case <-ticker.C:
            am.EvaluateAlerts(ctx)
        }
    }
}

func (am *alertManager) backupCheckLoop(ctx context.Context) {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    defer am.wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-am.stopCh:
            return
        case <-ticker.C:
            am.backupMonitor.RunDailyVerification(ctx)
        }
    }
}

func (am *alertManager) gcLoop(ctx context.Context) {
    ticker := time.NewTicker(am.config.GCInterval)
    defer ticker.Stop()
    defer am.wg.Done()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-am.stopCh:
            return
        case <-ticker.C:
            am.runGarbageCollection(ctx)
        }
    }
}

func (am *alertManager) runGarbageCollection(ctx context.Context) {
    cutoff := time.Now().Add(-am.config.RetentionPeriod)
    resolved, _ := am.repository.GetAlertsByTimeRange(ctx, time.Time{}, cutoff)
    
    for _, alert := range resolved {
        if alert.Status == StatusResolved {
            am.repository.DeleteAlert(ctx, alert.ID)
        }
    }
}

func (am *alertManager) loadAlertRules(ctx context.Context) ([]AlertRule, error) {
    rules := []AlertRule{
        {
            ID:          "high-error-rate",
            Name:        "HighErrorRate",
            Expr:        "rate(http_errors_total[5m]) > 0.05",
            For:         5 * time.Minute,
            Severity:    SeverityCritical,
            Type:        AlertTypeErrorRate,
            Enabled:     true,
            Annotations: map[string]string{"summary": "Error rate exceeds 5%"},
        },
        {
            ID:          "high-latency-p95",
            Name:        "HighLatencyP95",
            Expr:        "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2",
            For:         10 * time.Minute,
            Severity:    SeverityWarning,
            Type:        AlertTypeLatency,
            Enabled:     true,
            Annotations: map[string]string{"summary": "P95 latency exceeds 2s"},
        },
        {
            ID:          "service-down",
            Name:        "ServiceDown",
            Expr:        "up == 0",
            For:         2 * time.Minute,
            Severity:    SeverityCritical,
            Type:        AlertTypeUptime,
            Enabled:     true,
            Annotations: map[string]string{"summary": "Service is unreachable"},
        },
        {
            ID:          "concurrent-users-high",
            Name:        "HighConcurrentUsers",
            Expr:        "concurrent_users > 10000",
            For:         15 * time.Minute,
            Severity:    SeverityWarning,
            Type:        AlertTypeConcurrentUsers,
            Enabled:     true,
            Annotations: map[string]string{"summary": "Concurrent users approaching limit"},
        },
        {
            ID:          "backup-failed",
            Name:        "BackupFailed",
            Expr:        "backup_status == 0",
            For:         0,
            Severity:    SeverityCritical,
            Type:        AlertTypeBackupFailed,
            Enabled:     true,
            Annotations: map[string]string{"summary": "Daily backup failed"},
        },
        {
            ID:          "backup-restore-failed",
            Name:        "BackupRestoreFailed",
            Expr:        "backup_restore_test == 0",
            For:         0,
            Severity:    SeverityCritical,
            Type:        AlertTypeBackupRestore,
            Enabled:     true,
            Annotations: map[string]string{"summary": "Backup restore test failed"},
        },
    }
    return rules, nil
}

func (am *alertManager) processEvaluationResult(ctx context.Context, rule AlertRule, result EvaluationResult) {
    silences, _ := am.repository.GetActiveSilences(ctx)
    
    alert := Alert{
        ID:          generateFingerprint(rule, result),
        Name:        rule.Name,
        Description: rule.Annotations["summary"],
        Severity:    rule.Severity,
        Type:        rule.Type,
        Labels:      rule.Labels,
        Annotations: rule.Annotations,
        StartsAt:    result.Timestamp,
        Generator:   "AlertManager",
    }
    
    if IsAlertSilenced(alert, silences) {
        return
    }
    
    if result.IsFiring {
        if result.Duration >= rule.For {
            alert.Status = StatusFiring
            am.sendAlertNotifications(ctx, alert)
        } else {
            alert.Status = StatusPending
        }
    } else {
        alert.Status = StatusResolved
        endsAt := time.Now()
        alert.EndsAt = &endsAt
    }
    
    am.repository.SaveAlert(ctx, alert)
}

func (am *alertManager) sendAlertNotifications(ctx context.Context, alert Alert) {
    channels, _ := am.repository.GetNotificationChannels(ctx)
    
    for _, ch := range channels {
        if matchesLabels(ch.Labels, alert.Labels) {
            go func(channel NotificationChannel) {
                ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
                defer cancel()
                
                if err := am.notifier.Send(ctx, channel, alert); err != nil {
                    log.Printf("Notification failed for channel %s: %v", channel.ID, err)
                    am.handleNotificationFailure(ctx, channel, alert)
                }
            }(ch)
        }
    }
}

func (am *alertManager) handleNotificationFailure(ctx context.Context, channel NotificationChannel, alert Alert) {
    // Log failure, implement circuit breaker logic
    // Could trigger secondary notification channel
}

func generateID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateFingerprint(rule AlertRule, result EvaluationResult) string {
    data := fmt.Sprintf("%s:%s:%s:%v", rule.Name, rule.Severity, rule.Labels["service"], result.Timestamp.Unix())
    hash := sha256.Sum256([]byte(data))
    return fmt.Sprintf("%x", hash[:8])
}

func matchesLabels(channelLabels, alertLabels map[string]string) bool {
    for k, v := range channelLabels {
        if alertLabels[k] != v {
            return false
        }
    }
    return true
}
```

**Integration with GCP Cloud Monitoring:**

```go
type GCPAlertEvaluator struct {
    client    *monitoring.MetricClient
    projectID string
}

func (e *GCPAlertEvaluator) EvaluateRule(ctx context.Context, rule AlertRule) (EvaluationResult, error) {
    now := time.Now()
    
    req := &monitoringpb.QueryTimeSeriesRequest{
        Name:   fmt.Sprintf("projects/%s", e.projectID),
        Query:  rule.Expr,
        Period: durationpb.New(rule.Window),
    }
    
    response, err := e.client.QueryTimeSeries(ctx, req)
    if err != nil {
        return EvaluationResult{}, err
    }
    
    value := extractValue(response)
    isFiring := checkCondition(value, rule.Threshold)
    
    return EvaluationResult{
        IsFiring: isFiring,
        Value:    value,
        Timestamp: now,
        Duration: 0,
    }, nil
}
```

**Alert Configuration Loading:**

```go
func LoadAlertRulesFromConfig(configPath string) ([]AlertRule, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    
    var rules []AlertRule
    if err := json.Unmarshal(data, &rules); err != nil {
        return nil, err
    }
    
    return rules, nil
}
```
