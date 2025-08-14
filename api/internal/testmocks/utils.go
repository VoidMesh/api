// Package testmocks provides utilities for working with mocks in tests
package testmocks

import (
    "testing"
    
    "github.com/pashagolub/pgxmock/v4"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
)

// MockController is a convenience wrapper around gomock.Controller
type MockController struct {
    *gomock.Controller
}

// NewMockController creates a new mock controller for the given test
func NewMockController(t *testing.T) *MockController {
    ctrl := gomock.NewController(t)
    
    // Ensure controller finishes properly
    t.Cleanup(ctrl.Finish)
    
    return &MockController{Controller: ctrl}
}

// NewPgxMock creates a new pgxmock for database testing
func NewPgxMock(t *testing.T) pgxmock.PgxPoolIface {
    mock, err := pgxmock.NewPool()
    require.NoError(t, err, "Failed to create pgx mock")
    
    t.Cleanup(func() {
        mock.Close()
    })
    
    return mock
}

// AssertMockExpectations verifies that all mock expectations were met
func AssertMockExpectations(t *testing.T, mocks ...interface{}) {
    t.Helper()
    
    for _, mock := range mocks {
        if pgxMock, ok := mock.(pgxmock.PgxPoolIface); ok {
            require.NoError(t, pgxMock.ExpectationsWereMet(), "pgx mock expectations were not met")
        }
        // Add other mock types as needed
    }
}
