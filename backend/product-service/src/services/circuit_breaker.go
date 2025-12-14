package services

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	// StateClosed means requests are allowed
	StateClosed CircuitState = iota
	// StateOpen means requests are blocked
	StateOpen
	// StateHalfOpen means limited requests are allowed to test recovery
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for S3 operations
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	lastStateChange   time.Time
	failureThreshold  int           // Number of failures before opening
	successThreshold  int           // Number of successes in half-open before closing
	openTimeout       time.Duration // How long to wait before trying half-open
	halfOpenMaxCalls  int           // Max concurrent calls in half-open state
	halfOpenCallCount int           // Current concurrent calls in half-open
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		openTimeout:      openTimeout,
		halfOpenMaxCalls: 3, // Allow 3 concurrent test calls in half-open
		lastStateChange:  time.Now(),
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(operation func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := operation()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastStateChange) >= cb.openTimeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenCallCount = 0
			log.Info().Msg("Circuit breaker transitioning from OPEN to HALF-OPEN")
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited concurrent requests
		if cb.halfOpenCallCount < cb.halfOpenMaxCalls {
			cb.halfOpenCallCount++
			return true
		}
		return false

	default:
		return false
	}
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		cb.halfOpenCallCount--

		// If we've had enough successes, close the circuit
		if cb.successCount >= cb.successThreshold {
			cb.setState(StateClosed)
			cb.failureCount = 0
			cb.successCount = 0
			log.Info().
				Int("successes", cb.successThreshold).
				Msg("Circuit breaker transitioning from HALF-OPEN to CLOSED")
		}
	case StateClosed:
		// Reset failure count on success
		if cb.failureCount > 0 {
			cb.failureCount = 0
		}
	}
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()
	cb.failureCount++

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.setState(StateOpen)
			log.Warn().
				Int("failures", cb.failureCount).
				Dur("open_timeout", cb.openTimeout).
				Msg("Circuit breaker transitioning from CLOSED to OPEN")
		}

	case StateHalfOpen:
		// Any failure in half-open immediately re-opens the circuit
		cb.halfOpenCallCount--
		cb.setState(StateOpen)
		cb.successCount = 0
		log.Warn().
			Msg("Circuit breaker transitioning from HALF-OPEN to OPEN due to failure")
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(state CircuitState) {
	cb.state = state
	cb.lastStateChange = time.Now()
}

// GetState returns the current state (for monitoring/debugging)
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stateStr := "CLOSED"
	switch cb.state {
	case StateOpen:
		stateStr = "OPEN"
	case StateHalfOpen:
		stateStr = "HALF-OPEN"
	}

	stats := map[string]interface{}{
		"state":                stateStr,
		"failure_count":        cb.failureCount,
		"success_count":        cb.successCount,
		"failure_threshold":    cb.failureThreshold,
		"success_threshold":    cb.successThreshold,
		"open_timeout_seconds": cb.openTimeout.Seconds(),
		"last_state_change":    cb.lastStateChange,
		"seconds_since_change": time.Since(cb.lastStateChange).Seconds(),
	}

	if !cb.lastFailureTime.IsZero() {
		stats["last_failure_time"] = cb.lastFailureTime
		stats["seconds_since_failure"] = time.Since(cb.lastFailureTime).Seconds()
	}

	if cb.state == StateHalfOpen {
		stats["half_open_calls"] = cb.halfOpenCallCount
		stats["half_open_max_calls"] = cb.halfOpenMaxCalls
	}

	return stats
}

// Reset manually resets the circuit breaker (for testing/admin purposes)
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCallCount = 0
	cb.lastStateChange = time.Now()

	log.Info().Msg("Circuit breaker manually reset to CLOSED state")
}
