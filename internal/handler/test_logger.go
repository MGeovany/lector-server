package handler

import "pdf-text-reader/internal/domain"

// Mock logger used by handler package tests.
type MockHandlerLogger struct{}

func NewMockHandlerLogger() domain.Logger {
	return &MockHandlerLogger{}
}

func (l *MockHandlerLogger) Info(msg string, fields ...interface{})  {}
func (l *MockHandlerLogger) Error(msg string, err error, fields ...interface{}) {}
func (l *MockHandlerLogger) Debug(msg string, fields ...interface{}) {}
func (l *MockHandlerLogger) Warn(msg string, fields ...interface{})  {}

