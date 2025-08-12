# Adapter Authoring Guide

## Overview

This guide covers how to implement new adapters in Ephemos' hexagonal architecture, including creating ports, implementing adapters, writing tests, and avoiding common pitfalls.

## Quick Reference

```
Port (Interface)     → internal/core/ports/
Primary Adapter      → internal/adapters/primary/
Secondary Adapter    → internal/adapters/secondary/
Contract            → internal/contract/
Tests               → *_test.go files
```

## Table of Contents

1. [Creating a New Port (Interface)](#creating-a-new-port-interface)
2. [Implementing Primary Adapters](#implementing-primary-adapters)
3. [Implementing Secondary Adapters](#implementing-secondary-adapters)
4. [Required Testing Patterns](#required-testing-patterns)
5. [Common Pitfalls and Solutions](#common-pitfalls-and-solutions)
6. [Real Examples Walkthrough](#real-examples-walkthrough)
7. [Architecture Compliance Checklist](#architecture-compliance-checklist)

## Creating a New Port (Interface)

### 1. Define the Port Interface

Create your port interface in `internal/core/ports/`:

```go
// internal/core/ports/messaging.go
package ports

import (
    "context"
    "github.com/sufield/ephemos/internal/core/domain"
)

// MessagePublisher defines the interface for publishing messages
type MessagePublisher interface {
    // Publish sends a message to the specified topic
    Publish(ctx context.Context, topic string, message []byte) error
    
    // PublishWithHeaders sends a message with custom headers
    PublishWithHeaders(ctx context.Context, req PublishRequest) error
    
    // Close gracefully shuts down the publisher
    Close() error
    
    // Health returns the current health status
    Health() error
}

// MessageSubscriber defines the interface for consuming messages
type MessageSubscriber interface {
    // Subscribe starts consuming messages from specified topics
    Subscribe(ctx context.Context, topics []string, handler MessageHandler) error
    
    // Unsubscribe stops consuming from specified topics
    Unsubscribe(topics []string) error
    
    // Close gracefully shuts down the subscriber
    Close() error
}

// MessageHandler processes incoming messages
type MessageHandler interface {
    Handle(ctx context.Context, message Message) error
}

// PublishRequest contains message publishing parameters
type PublishRequest struct {
    Topic   string
    Message []byte
    Headers map[string]string
}

// Message represents a received message
type Message struct {
    Topic     string
    Payload   []byte
    Headers   map[string]string
    Timestamp time.Time
    ID        string
}
```

### 2. Add Domain Types (if needed)

If your port needs domain-specific types, add them to `internal/core/domain/`:

```go
// internal/core/domain/messaging.go
package domain

import "time"

// MessageConfig contains messaging configuration
type MessageConfig struct {
    Brokers          []string      `yaml:"brokers" json:"brokers"`
    SecurityProtocol string        `yaml:"security_protocol" json:"security_protocol"`
    RetryAttempts    int           `yaml:"retry_attempts" json:"retry_attempts"`
    Timeout          time.Duration `yaml:"timeout" json:"timeout"`
}

// MessageStats contains messaging statistics
type MessageStats struct {
    Published    int64     `json:"published"`
    Consumed     int64     `json:"consumed"`
    Errors       int64     `json:"errors"`
    LastActivity time.Time `json:"last_activity"`
}
```

### 3. Port Testing Template

```go
// internal/core/ports/messaging_test.go
package ports_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/sufield/ephemos/internal/core/ports"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// MockMessagePublisher implements ports.MessagePublisher for testing
type MockMessagePublisher struct {
    published []ports.PublishRequest
    closed    bool
    healthy   bool
}

func (m *MockMessagePublisher) Publish(ctx context.Context, topic string, message []byte) error {
    return m.PublishWithHeaders(ctx, ports.PublishRequest{
        Topic:   topic,
        Message: message,
    })
}

func (m *MockMessagePublisher) PublishWithHeaders(ctx context.Context, req ports.PublishRequest) error {
    if !m.healthy {
        return errors.New("publisher unhealthy")
    }
    m.published = append(m.published, req)
    return nil
}

func (m *MockMessagePublisher) Close() error {
    m.closed = true
    return nil
}

func (m *MockMessagePublisher) Health() error {
    if !m.healthy {
        return errors.New("publisher unhealthy")
    }
    return nil
}

func TestMessagePublisherInterface(t *testing.T) {
    t.Run("publish message", func(t *testing.T) {
        pub := &MockMessagePublisher{healthy: true}
        
        err := pub.Publish(context.Background(), "test-topic", []byte("hello"))
        
        assert.NoError(t, err)
        assert.Len(t, pub.published, 1)
        assert.Equal(t, "test-topic", pub.published[0].Topic)
        assert.Equal(t, []byte("hello"), pub.published[0].Message)
    })
    
    t.Run("health check", func(t *testing.T) {
        pub := &MockMessagePublisher{healthy: true}
        assert.NoError(t, pub.Health())
        
        pub.healthy = false
        assert.Error(t, pub.Health())
    })
    
    t.Run("close publisher", func(t *testing.T) {
        pub := &MockMessagePublisher{healthy: true}
        
        err := pub.Close()
        
        assert.NoError(t, err)
        assert.True(t, pub.closed)
    })
}
```

## Implementing Primary Adapters

Primary adapters drive the application (inbound). They receive external requests and translate them to core business operations.

### 1. Primary Adapter Structure

```go
// internal/adapters/primary/messaging/rest_handler.go
package messaging

import (
    "encoding/json"
    "net/http"
    
    "github.com/sufield/ephemos/internal/core/ports"
    "github.com/sufield/ephemos/internal/core/services"
)

// RESTHandler provides HTTP endpoints for messaging
type RESTHandler struct {
    messagingService ports.MessagePublisher
    logger          ports.Logger
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(
    messagingService ports.MessagePublisher,
    logger ports.Logger,
) *RESTHandler {
    return &RESTHandler{
        messagingService: messagingService,
        logger:          logger,
    }
}

// PublishMessage handles POST /api/messages
func (h *RESTHandler) PublishMessage(w http.ResponseWriter, r *http.Request) {
    var req PublishMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    
    if err := h.validatePublishRequest(req); err != nil {
        h.writeError(w, http.StatusBadRequest, err.Error())
        return
    }
    
    publishReq := ports.PublishRequest{
        Topic:   req.Topic,
        Message: []byte(req.Message),
        Headers: req.Headers,
    }
    
    if err := h.messagingService.PublishWithHeaders(r.Context(), publishReq); err != nil {
        h.logger.Error("Failed to publish message", "error", err, "topic", req.Topic)
        h.writeError(w, http.StatusInternalServerError, "failed to publish message")
        return
    }
    
    h.writeResponse(w, http.StatusOK, PublishMessageResponse{
        Success: true,
        MessageID: generateMessageID(),
    })
}

// Health handles GET /health
func (h *RESTHandler) Health(w http.ResponseWriter, r *http.Request) {
    if err := h.messagingService.Health(); err != nil {
        h.writeError(w, http.StatusServiceUnavailable, "messaging service unhealthy")
        return
    }
    
    h.writeResponse(w, http.StatusOK, map[string]string{
        "status": "healthy",
    })
}

// Request/Response types
type PublishMessageRequest struct {
    Topic   string            `json:"topic"`
    Message string            `json:"message"`
    Headers map[string]string `json:"headers,omitempty"`
}

type PublishMessageResponse struct {
    Success   bool   `json:"success"`
    MessageID string `json:"message_id"`
}

func (h *RESTHandler) validatePublishRequest(req PublishMessageRequest) error {
    if req.Topic == "" {
        return errors.New("topic is required")
    }
    if req.Message == "" {
        return errors.New("message is required")
    }
    return nil
}

func (h *RESTHandler) writeError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error": message,
    })
}

func (h *RESTHandler) writeResponse(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func generateMessageID() string {
    return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
```

### 2. Primary Adapter Testing

```go
// internal/adapters/primary/messaging/rest_handler_test.go
package messaging_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/sufield/ephemos/internal/adapters/primary/messaging"
    "github.com/sufield/ephemos/internal/core/ports"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRESTHandler_PublishMessage(t *testing.T) {
    t.Run("successful publish", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: true}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        reqBody := messaging.PublishMessageRequest{
            Topic:   "test-topic",
            Message: "hello world",
            Headers: map[string]string{"key": "value"},
        }
        
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)
        
        req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        recorder := httptest.NewRecorder()
        
        handler.PublishMessage(recorder, req)
        
        assert.Equal(t, http.StatusOK, recorder.Code)
        
        var response messaging.PublishMessageResponse
        err = json.NewDecoder(recorder.Body).Decode(&response)
        require.NoError(t, err)
        assert.True(t, response.Success)
        assert.NotEmpty(t, response.MessageID)
        
        // Verify the publisher was called correctly
        assert.Len(t, mockPublisher.published, 1)
        assert.Equal(t, "test-topic", mockPublisher.published[0].Topic)
        assert.Equal(t, []byte("hello world"), mockPublisher.published[0].Message)
        assert.Equal(t, map[string]string{"key": "value"}, mockPublisher.published[0].Headers)
    })
    
    t.Run("invalid request body", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: true}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader([]byte("invalid json")))
        req.Header.Set("Content-Type", "application/json")
        recorder := httptest.NewRecorder()
        
        handler.PublishMessage(recorder, req)
        
        assert.Equal(t, http.StatusBadRequest, recorder.Code)
        
        var response map[string]string
        err := json.NewDecoder(recorder.Body).Decode(&response)
        require.NoError(t, err)
        assert.Contains(t, response["error"], "invalid request body")
    })
    
    t.Run("missing required fields", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: true}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        reqBody := messaging.PublishMessageRequest{
            // Missing topic and message
        }
        
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)
        
        req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        recorder := httptest.NewRecorder()
        
        handler.PublishMessage(recorder, req)
        
        assert.Equal(t, http.StatusBadRequest, recorder.Code)
    })
    
    t.Run("publisher service error", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: false}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        reqBody := messaging.PublishMessageRequest{
            Topic:   "test-topic",
            Message: "hello world",
        }
        
        body, err := json.Marshal(reqBody)
        require.NoError(t, err)
        
        req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        recorder := httptest.NewRecorder()
        
        handler.PublishMessage(recorder, req)
        
        assert.Equal(t, http.StatusInternalServerError, recorder.Code)
        
        // Verify error was logged
        assert.Len(t, mockLogger.entries, 1)
        assert.Equal(t, "error", mockLogger.entries[0].Level)
    })
}

func TestRESTHandler_Health(t *testing.T) {
    t.Run("healthy service", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: true}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        req := httptest.NewRequest("GET", "/health", nil)
        recorder := httptest.NewRecorder()
        
        handler.Health(recorder, req)
        
        assert.Equal(t, http.StatusOK, recorder.Code)
        
        var response map[string]string
        err := json.NewDecoder(recorder.Body).Decode(&response)
        require.NoError(t, err)
        assert.Equal(t, "healthy", response["status"])
    })
    
    t.Run("unhealthy service", func(t *testing.T) {
        mockPublisher := &MockMessagePublisher{healthy: false}
        mockLogger := &MockLogger{}
        handler := messaging.NewRESTHandler(mockPublisher, mockLogger)
        
        req := httptest.NewRequest("GET", "/health", nil)
        recorder := httptest.NewRecorder()
        
        handler.Health(recorder, req)
        
        assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
    })
}
```

## Implementing Secondary Adapters

Secondary adapters are driven by the application (outbound). They implement the ports that core services depend on.

### 1. Secondary Adapter Structure

```go
// internal/adapters/secondary/messaging/kafka_publisher.go
package messaging

import (
    "context"
    "fmt"
    "time"
    
    "github.com/IBM/sarama"
    "github.com/sufield/ephemos/internal/core/domain"
    "github.com/sufield/ephemos/internal/core/ports"
)

// KafkaPublisher implements ports.MessagePublisher using Apache Kafka
type KafkaPublisher struct {
    producer sarama.SyncProducer
    config   domain.MessageConfig
    logger   ports.Logger
    closed   bool
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(config domain.MessageConfig, logger ports.Logger) (*KafkaPublisher, error) {
    saramaConfig := sarama.NewConfig()
    saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
    saramaConfig.Producer.Retry.Max = config.RetryAttempts
    saramaConfig.Producer.Return.Successes = true
    saramaConfig.Net.DialTimeout = config.Timeout
    
    // Configure security if specified
    if config.SecurityProtocol == "SASL_SSL" {
        saramaConfig.Net.SASL.Enable = true
        saramaConfig.Net.TLS.Enable = true
    }
    
    producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create kafka producer: %w", err)
    }
    
    return &KafkaPublisher{
        producer: producer,
        config:   config,
        logger:   logger,
    }, nil
}

// Publish implements ports.MessagePublisher
func (k *KafkaPublisher) Publish(ctx context.Context, topic string, message []byte) error {
    return k.PublishWithHeaders(ctx, ports.PublishRequest{
        Topic:   topic,
        Message: message,
    })
}

// PublishWithHeaders implements ports.MessagePublisher
func (k *KafkaPublisher) PublishWithHeaders(ctx context.Context, req ports.PublishRequest) error {
    if k.closed {
        return errors.New("publisher is closed")
    }
    
    msg := &sarama.ProducerMessage{
        Topic: req.Topic,
        Value: sarama.ByteEncoder(req.Message),
    }
    
    // Add headers if provided
    if len(req.Headers) > 0 {
        msg.Headers = make([]sarama.RecordHeader, 0, len(req.Headers))
        for key, value := range req.Headers {
            msg.Headers = append(msg.Headers, sarama.RecordHeader{
                Key:   []byte(key),
                Value: []byte(value),
            })
        }
    }
    
    // Add metadata headers
    msg.Headers = append(msg.Headers,
        sarama.RecordHeader{Key: []byte("timestamp"), Value: []byte(time.Now().Format(time.RFC3339))},
        sarama.RecordHeader{Key: []byte("source"), Value: []byte("ephemos")},
    )
    
    partition, offset, err := k.producer.SendMessage(msg)
    if err != nil {
        k.logger.Error("Failed to publish message", 
            "error", err, 
            "topic", req.Topic,
            "message_size", len(req.Message))
        return fmt.Errorf("failed to publish message to topic %s: %w", req.Topic, err)
    }
    
    k.logger.Debug("Message published successfully", 
        "topic", req.Topic, 
        "partition", partition, 
        "offset", offset,
        "message_size", len(req.Message))
    
    return nil
}

// Close implements ports.MessagePublisher
func (k *KafkaPublisher) Close() error {
    if k.closed {
        return nil
    }
    
    k.closed = true
    return k.producer.Close()
}

// Health implements ports.MessagePublisher
func (k *KafkaPublisher) Health() error {
    if k.closed {
        return errors.New("publisher is closed")
    }
    
    // Try to get metadata as a health check
    _, err := k.producer.GetMetadata()
    if err != nil {
        return fmt.Errorf("kafka health check failed: %w", err)
    }
    
    return nil
}

// GetStats returns publishing statistics
func (k *KafkaPublisher) GetStats() domain.MessageStats {
    // This would typically track real statistics
    return domain.MessageStats{
        Published:    0, // Would be tracked in real implementation
        Consumed:     0,
        Errors:       0,
        LastActivity: time.Now(),
    }
}
```

### 2. Secondary Adapter Testing

```go
// internal/adapters/secondary/messaging/kafka_publisher_test.go
package messaging_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/sufield/ephemos/internal/adapters/secondary/messaging"
    "github.com/sufield/ephemos/internal/core/domain"
    "github.com/sufield/ephemos/internal/core/ports"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Integration test (requires running Kafka)
func TestKafkaPublisher_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // These tests require a running Kafka instance
    config := domain.MessageConfig{
        Brokers:       []string{"localhost:9092"},
        RetryAttempts: 3,
        Timeout:       5 * time.Second,
    }
    
    logger := &MockLogger{}
    
    publisher, err := messaging.NewKafkaPublisher(config, logger)
    require.NoError(t, err)
    defer publisher.Close()
    
    t.Run("publish message successfully", func(t *testing.T) {
        err := publisher.Publish(context.Background(), "test-topic", []byte("test message"))
        assert.NoError(t, err)
    })
    
    t.Run("publish with headers", func(t *testing.T) {
        req := ports.PublishRequest{
            Topic:   "test-topic-headers",
            Message: []byte("test message with headers"),
            Headers: map[string]string{
                "correlation-id": "12345",
                "user-id":        "test-user",
            },
        }
        
        err := publisher.PublishWithHeaders(context.Background(), req)
        assert.NoError(t, err)
    })
    
    t.Run("health check", func(t *testing.T) {
        err := publisher.Health()
        assert.NoError(t, err)
    })
}

// Unit tests with mocked Kafka producer
func TestKafkaPublisher_Unit(t *testing.T) {
    t.Run("publish to closed publisher fails", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers: []string{"localhost:9092"},
        }
        logger := &MockLogger{}
        
        publisher, err := messaging.NewKafkaPublisher(config, logger)
        require.NoError(t, err)
        
        // Close the publisher
        err = publisher.Close()
        require.NoError(t, err)
        
        // Try to publish - should fail
        err = publisher.Publish(context.Background(), "test-topic", []byte("test"))
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "publisher is closed")
    })
    
    t.Run("health check on closed publisher fails", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers: []string{"localhost:9092"},
        }
        logger := &MockLogger{}
        
        publisher, err := messaging.NewKafkaPublisher(config, logger)
        require.NoError(t, err)
        
        // Close the publisher
        err = publisher.Close()
        require.NoError(t, err)
        
        // Health check should fail
        err = publisher.Health()
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "publisher is closed")
    })
}

// Benchmark tests
func BenchmarkKafkaPublisher_Publish(b *testing.B) {
    config := domain.MessageConfig{
        Brokers:       []string{"localhost:9092"},
        RetryAttempts: 1,
        Timeout:       time.Second,
    }
    
    logger := &MockLogger{}
    publisher, err := messaging.NewKafkaPublisher(config, logger)
    require.NoError(b, err)
    defer publisher.Close()
    
    message := []byte("benchmark test message")
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := publisher.Publish(context.Background(), "benchmark-topic", message)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Required Testing Patterns

### 1. Contract Testing

Test that adapters properly implement the port interfaces:

```go
// internal/adapters/secondary/messaging/contract_test.go
package messaging_test

import (
    "testing"
    
    "github.com/sufield/ephemos/internal/adapters/secondary/messaging"
    "github.com/sufield/ephemos/internal/core/domain"
    "github.com/sufield/ephemos/internal/core/ports"
)

func TestKafkaPublisher_ImplementsMessagePublisher(t *testing.T) {
    config := domain.MessageConfig{
        Brokers: []string{"localhost:9092"},
    }
    logger := &MockLogger{}
    
    publisher, err := messaging.NewKafkaPublisher(config, logger)
    require.NoError(t, err)
    defer publisher.Close()
    
    // Verify it implements the interface
    var _ ports.MessagePublisher = publisher
    
    // Test all interface methods
    t.Run("all methods callable", func(t *testing.T) {
        ctx := context.Background()
        
        // These should not panic
        _ = publisher.Publish(ctx, "test", []byte("test"))
        _ = publisher.PublishWithHeaders(ctx, ports.PublishRequest{})
        _ = publisher.Close()
        _ = publisher.Health()
    })
}
```

### 2. Error Handling Testing

```go
func TestKafkaPublisher_ErrorHandling(t *testing.T) {
    t.Run("connection failure scenarios", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers: []string{"invalid-broker:9092"},
            Timeout: time.Millisecond * 100,
        }
        logger := &MockLogger{}
        
        _, err := messaging.NewKafkaPublisher(config, logger)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "failed to create kafka producer")
    })
    
    t.Run("publish to invalid topic", func(t *testing.T) {
        // Test with mock that simulates broker errors
        // Implementation would use testify/mock or similar
    })
    
    t.Run("context cancellation handling", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers: []string{"localhost:9092"},
        }
        logger := &MockLogger{}
        
        publisher, err := messaging.NewKafkaPublisher(config, logger)
        require.NoError(t, err)
        defer publisher.Close()
        
        ctx, cancel := context.WithCancel(context.Background())
        cancel() // Cancel immediately
        
        err = publisher.Publish(ctx, "test-topic", []byte("test"))
        // Should handle cancellation gracefully
        assert.Error(t, err)
    })
}
```

### 3. Configuration Testing

```go
func TestKafkaPublisher_Configuration(t *testing.T) {
    t.Run("security configuration", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers:          []string{"localhost:9092"},
            SecurityProtocol: "SASL_SSL",
        }
        logger := &MockLogger{}
        
        publisher, err := messaging.NewKafkaPublisher(config, logger)
        if err != nil {
            t.Skip("Kafka not available for security test")
        }
        defer publisher.Close()
        
        // Verify security settings are applied
        // This would typically require inspection of internal config
    })
    
    t.Run("retry configuration", func(t *testing.T) {
        config := domain.MessageConfig{
            Brokers:       []string{"localhost:9092"},
            RetryAttempts: 5,
        }
        logger := &MockLogger{}
        
        publisher, err := messaging.NewKafkaPublisher(config, logger)
        require.NoError(t, err)
        defer publisher.Close()
        
        // Test that retry configuration is respected
        // Would require mocking to verify retry behavior
    })
}
```

### 4. Performance Testing

```go
func TestKafkaPublisher_Performance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping performance test in short mode")
    }
    
    config := domain.MessageConfig{
        Brokers: []string{"localhost:9092"},
    }
    logger := &MockLogger{}
    
    publisher, err := messaging.NewKafkaPublisher(config, logger)
    require.NoError(t, err)
    defer publisher.Close()
    
    t.Run("concurrent publishing", func(t *testing.T) {
        const numGoroutines = 10
        const messagesPerGoroutine = 100
        
        var wg sync.WaitGroup
        errorsChan := make(chan error, numGoroutines*messagesPerGoroutine)
        
        for i := 0; i < numGoroutines; i++ {
            wg.Add(1)
            go func(workerID int) {
                defer wg.Done()
                for j := 0; j < messagesPerGoroutine; j++ {
                    message := fmt.Sprintf("worker-%d-message-%d", workerID, j)
                    err := publisher.Publish(context.Background(), "perf-test", []byte(message))
                    if err != nil {
                        errorsChan <- err
                    }
                }
            }(i)
        }
        
        wg.Wait()
        close(errorsChan)
        
        errors := make([]error, 0)
        for err := range errorsChan {
            errors = append(errors, err)
        }
        
        assert.Empty(t, errors, "Expected no errors during concurrent publishing")
    })
}
```

## Common Pitfalls and Solutions

### 1. ❌ Pitfall: Direct Adapter Dependencies

**Wrong:**
```go
// DON'T: Primary adapter importing secondary adapter directly
package api

import (
    "github.com/sufield/ephemos/internal/adapters/secondary/messaging" // ❌ WRONG
)

func (h *Handler) SendNotification() {
    kafka := messaging.NewKafkaPublisher(...) // ❌ WRONG
}
```

**✅ Correct:**
```go
// DO: Use dependency injection through ports
package api

import (
    "github.com/sufield/ephemos/internal/core/ports" // ✅ CORRECT
)

type Handler struct {
    publisher ports.MessagePublisher // ✅ CORRECT - depends on interface
}

func NewHandler(publisher ports.MessagePublisher) *Handler {
    return &Handler{publisher: publisher}
}
```

### 2. ❌ Pitfall: Business Logic in Adapters

**Wrong:**
```go
// DON'T: Business logic in adapter
func (k *KafkaPublisher) PublishOrderConfirmation(order Order) error {
    // ❌ WRONG: Business logic about order processing
    if order.Amount > 1000 {
        // Send to priority queue
        return k.Publish("priority-orders", order.ToJSON())
    }
    return k.Publish("regular-orders", order.ToJSON())
}
```

**✅ Correct:**
```go
// DO: Keep adapters focused on technical concerns
func (k *KafkaPublisher) Publish(ctx context.Context, topic string, message []byte) error {
    // ✅ CORRECT: Only handle Kafka-specific concerns
    return k.producer.SendMessage(&sarama.ProducerMessage{
        Topic: topic,
        Value: sarama.ByteEncoder(message),
    })
}

// Business logic belongs in core services
func (s *OrderService) ProcessOrder(order Order) error {
    topic := "regular-orders"
    if order.Amount > 1000 {
        topic = "priority-orders" // ✅ Business decision in service
    }
    
    return s.publisher.Publish(ctx, topic, order.ToJSON())
}
```

### 3. ❌ Pitfall: Inadequate Error Handling

**Wrong:**
```go
// DON'T: Swallow errors or provide unhelpful error messages
func (k *KafkaPublisher) Publish(topic string, message []byte) error {
    _, _, err := k.producer.SendMessage(msg)
    return err // ❌ WRONG: No context about what failed
}
```

**✅ Correct:**
```go
// DO: Provide context and wrap errors appropriately
func (k *KafkaPublisher) Publish(topic string, message []byte) error {
    msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(message)}
    
    partition, offset, err := k.producer.SendMessage(msg)
    if err != nil {
        // ✅ CORRECT: Wrap with context
        return fmt.Errorf("failed to publish message to topic %s: %w", topic, err)
    }
    
    k.logger.Debug("Message published", "topic", topic, "partition", partition, "offset", offset)
    return nil
}
```

### 4. ❌ Pitfall: Ignoring Context Cancellation

**Wrong:**
```go
// DON'T: Ignore context cancellation
func (k *KafkaPublisher) Publish(ctx context.Context, topic string, message []byte) error {
    // ❌ WRONG: Ignore ctx, operation might block indefinitely
    return k.producer.SendMessage(msg)
}
```

**✅ Correct:**
```go
// DO: Respect context cancellation and timeouts
func (k *KafkaPublisher) Publish(ctx context.Context, topic string, message []byte) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Create a channel to handle the async operation
    resultChan := make(chan error, 1)
    
    go func() {
        _, _, err := k.producer.SendMessage(msg)
        resultChan <- err
    }()
    
    select {
    case err := <-resultChan:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 5. ❌ Pitfall: Resource Leaks

**Wrong:**
```go
// DON'T: Forget to clean up resources
func (k *KafkaPublisher) Close() error {
    // ❌ WRONG: No cleanup, no idempotency check
    return k.producer.Close()
}
```

**✅ Correct:**
```go
// DO: Implement proper resource cleanup
func (k *KafkaPublisher) Close() error {
    k.mutex.Lock()
    defer k.mutex.Unlock()
    
    if k.closed {
        return nil // ✅ CORRECT: Idempotent
    }
    
    k.closed = true
    
    // ✅ CORRECT: Stop background goroutines
    if k.healthCheckCancel != nil {
        k.healthCheckCancel()
    }
    
    // ✅ CORRECT: Close resources
    if err := k.producer.Close(); err != nil {
        return fmt.Errorf("failed to close kafka producer: %w", err)
    }
    
    return nil
}
```

### 6. ❌ Pitfall: Poor Testability

**Wrong:**
```go
// DON'T: Hard-coded dependencies that can't be mocked
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
    // ❌ WRONG: Creates real producer, can't be tested without Kafka
    producer, _ := sarama.NewSyncProducer(brokers, sarama.NewConfig())
    return &KafkaPublisher{producer: producer}
}
```

**✅ Correct:**
```go
// DO: Accept interfaces for testability
type SaramaProducer interface {
    SendMessage(*sarama.ProducerMessage) (int32, int64, error)
    Close() error
}

type KafkaPublisher struct {
    producer SaramaProducer // ✅ CORRECT: Interface for mocking
}

func NewKafkaPublisher(producer SaramaProducer) *KafkaPublisher {
    return &KafkaPublisher{producer: producer}
}

// Factory function for real usage
func NewKafkaPublisherFromConfig(config domain.MessageConfig) (*KafkaPublisher, error) {
    producer, err := sarama.NewSyncProducer(config.Brokers, sarama.NewConfig())
    if err != nil {
        return nil, err
    }
    return NewKafkaPublisher(producer), nil
}
```

## Real Examples Walkthrough

### Adding a WebSocket Transport Adapter

Let's walk through adding WebSocket support as a real example:

#### 1. Define the Port

```go
// internal/core/ports/websocket.go
package ports

type WebSocketServer interface {
    Start(ctx context.Context, addr string) error
    Stop(ctx context.Context) error
    Broadcast(message []byte) error
    SendToClient(clientID string, message []byte) error
    RegisterHandler(event string, handler WebSocketHandler) error
}

type WebSocketHandler interface {
    Handle(ctx context.Context, conn WebSocketConnection, message []byte) error
}

type WebSocketConnection interface {
    ID() string
    Send(message []byte) error
    Close() error
    RemoteAddr() string
}
```

#### 2. Implement Secondary Adapter

```go
// internal/adapters/secondary/websocket/gorilla_server.go
package websocket

import (
    "github.com/gorilla/websocket"
    "github.com/sufield/ephemos/internal/core/ports"
)

type GorillaWebSocketServer struct {
    upgrader websocket.Upgrader
    handlers map[string]ports.WebSocketHandler
    clients  sync.Map // map[string]*GorillaConnection
    server   *http.Server
    logger   ports.Logger
}

func NewGorillaWebSocketServer(logger ports.Logger) *GorillaWebSocketServer {
    return &GorillaWebSocketServer{
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
        handlers: make(map[string]ports.WebSocketHandler),
        logger:   logger,
    }
}

func (g *GorillaWebSocketServer) Start(ctx context.Context, addr string) error {
    mux := http.NewServeMux()
    mux.HandleFunc("/ws", g.handleWebSocket)
    
    g.server = &http.Server{
        Addr:    addr,
        Handler: mux,
    }
    
    return g.server.ListenAndServe()
}

// ... rest of implementation
```

#### 3. Create Primary Adapter

```go
// internal/adapters/primary/websocket/handler.go
package websocket

type WebSocketPrimaryAdapter struct {
    server  ports.WebSocketServer
    service ports.MessageService
    logger  ports.Logger
}

func (w *WebSocketPrimaryAdapter) HandleChatMessage(ctx context.Context, conn ports.WebSocketConnection, message []byte) error {
    // Validate message
    chatMsg, err := w.parseMessage(message)
    if err != nil {
        return conn.Send([]byte(`{"error": "invalid message format"}`))
    }
    
    // Use core service
    result, err := w.service.ProcessChatMessage(ctx, chatMsg)
    if err != nil {
        w.logger.Error("Failed to process chat message", "error", err)
        return conn.Send([]byte(`{"error": "processing failed"}`))
    }
    
    // Broadcast to all clients
    return w.server.Broadcast(result.ToJSON())
}
```

#### 4. Write Comprehensive Tests

```go
// internal/adapters/secondary/websocket/gorilla_server_test.go
func TestGorillaWebSocketServer(t *testing.T) {
    t.Run("client connection and messaging", func(t *testing.T) {
        server := NewGorillaWebSocketServer(&MockLogger{})
        
        // Start server on random port
        listener, err := net.Listen("tcp", ":0")
        require.NoError(t, err)
        defer listener.Close()
        
        addr := listener.Addr().String()
        go server.Start(context.Background(), addr)
        
        // Connect WebSocket client
        wsURL := "ws://" + addr + "/ws"
        conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
        require.NoError(t, err)
        defer conn.Close()
        
        // Test messaging
        testMessage := []byte("hello websocket")
        err = conn.WriteMessage(websocket.TextMessage, testMessage)
        assert.NoError(t, err)
        
        // Verify message received
        _, receivedMessage, err := conn.ReadMessage()
        assert.NoError(t, err)
        assert.Equal(t, testMessage, receivedMessage)
    })
}
```

## Architecture Compliance Checklist

### ✅ Port Design Checklist

- [ ] Interface is defined in `internal/core/ports/`
- [ ] No external library types in interface signatures
- [ ] Context parameter for all operations that might block
- [ ] Error return values for all fallible operations
- [ ] Graceful shutdown method (`Close()` or `Stop()`)
- [ ] Health check method if applicable
- [ ] Interface is focused (single responsibility)
- [ ] Domain types used instead of primitive obsession

### ✅ Primary Adapter Checklist

- [ ] Located in `internal/adapters/primary/`
- [ ] Only imports from `internal/core/ports` and `internal/core/services`
- [ ] Handles protocol-specific concerns (HTTP, gRPC, CLI)
- [ ] Validates input and converts to domain types
- [ ] Proper error handling and logging
- [ ] Does not contain business logic
- [ ] Comprehensive tests including error scenarios
- [ ] Graceful degradation on service failures

### ✅ Secondary Adapter Checklist

- [ ] Located in `internal/adapters/secondary/`
- [ ] Implements port interface correctly
- [ ] Only imports from `internal/core/ports` and `internal/core/domain`
- [ ] Handles external system integration
- [ ] Proper resource management (connections, files, etc.)
- [ ] Context cancellation support
- [ ] Retry logic with exponential backoff where appropriate
- [ ] Health checking capability
- [ ] Configuration through domain types
- [ ] Both unit and integration tests

### ✅ Testing Checklist

- [ ] Contract tests verify interface implementation
- [ ] Unit tests with mocked dependencies
- [ ] Integration tests with real external systems
- [ ] Error handling tests for all failure modes
- [ ] Performance/benchmark tests for critical paths
- [ ] Concurrent usage tests
- [ ] Resource leak tests (goroutines, connections, files)
- [ ] Configuration validation tests
- [ ] Context cancellation tests

### ✅ General Compliance Checklist

- [ ] No circular dependencies
- [ ] No core → adapter imports
- [ ] Proper error wrapping with context
- [ ] Logging at appropriate levels
- [ ] Metrics/observability hooks
- [ ] Graceful shutdown support
- [ ] Configuration externalization
- [ ] Documentation and examples
- [ ] Security considerations addressed
- [ ] Performance implications considered

## Tools and Commands

### Check Architecture Compliance

```bash
# Check for illegal imports from core to adapters
go list -f '{{.ImportPath}} {{.Imports}}' ./internal/core/... | grep -E "(adapter|external)"

# Verify no circular dependencies  
go mod graph | grep internal | sort | uniq -c | sort -rn

# Run architecture tests
go test ./internal/core/ports/architecture_test.go

# Generate dependency graph
go mod graph | grep internal > deps.txt
```

### Testing Commands

```bash
# Run all tests
go test ./internal/adapters/...

# Run only unit tests (exclude integration)
go test -short ./internal/adapters/...

# Run with race detection
go test -race ./internal/adapters/...

# Run benchmarks
go test -bench=. ./internal/adapters/...

# Generate test coverage
go test -coverprofile=coverage.out ./internal/adapters/...
go tool cover -html=coverage.out
```

### Code Quality Checks

```bash
# Lint the code
golangci-lint run ./internal/adapters/...

# Check for vulnerable dependencies
govulncheck ./internal/adapters/...

# Format code
gofumpt -w ./internal/adapters/

# Generate mocks for testing
mockgen -source=internal/core/ports/messaging.go -destination=internal/mocks/mock_messaging.go
```

---

*This guide covers the essential patterns for implementing clean adapters in Ephemos' hexagonal architecture. Follow these patterns to ensure your adapters are maintainable, testable, and compliant with the architecture principles.*

*Last Updated: December 2024*
*Guide Version: 1.0*