package mocks

import (
	"context"

	"github.com/pos/user-service/src/utils"
)

// MockAuditPublisher is a mock implementation of utils.AuditPublisherInterface for testing
// It simulates audit event publishing without requiring Kafka
type MockAuditPublisher struct {
	PublishFunc      func(ctx context.Context, event *utils.AuditEvent) error
	PublishBatchFunc func(ctx context.Context, events []*utils.AuditEvent) error
	CloseFunc        func() error
}

// Publish simulates publishing a single audit event
func (m *MockAuditPublisher) Publish(ctx context.Context, event *utils.AuditEvent) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event)
	}
	// Default mock behavior - do nothing
	return nil
}

// PublishBatch simulates publishing a batch of audit events
func (m *MockAuditPublisher) PublishBatch(ctx context.Context, events []*utils.AuditEvent) error {
	if m.PublishBatchFunc != nil {
		return m.PublishBatchFunc(ctx, events)
	}
	// Default mock behavior - do nothing
	return nil
}

// Close simulates closing the publisher
func (m *MockAuditPublisher) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	// Default mock behavior - do nothing
	return nil
}

// NoOpAuditPublisher is a no-op audit publisher for testing
// All operations succeed without side effects
type NoOpAuditPublisher struct{}

func (n *NoOpAuditPublisher) Publish(ctx context.Context, event *utils.AuditEvent) error {
	return nil
}

func (n *NoOpAuditPublisher) PublishBatch(ctx context.Context, events []*utils.AuditEvent) error {
	return nil
}

func (n *NoOpAuditPublisher) Close() error {
	return nil
}
