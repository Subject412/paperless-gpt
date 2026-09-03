package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestJobLoggingTruncation(t *testing.T) {
	buf := &bytes.Buffer{}
	testLogger := logrus.New()
	testLogger.SetOutput(buf)
	testLogger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	// Backup original logger and restore after test
	origLogger := logger
	logger = testLogger
	defer func() { logger = origLogger }()

	job := &Job{
		ID:         "test-job-123",
		DocumentID: 42,
		Status:     "pending",
		Result:     strings.Repeat("Long OCR Result ", 1000), // Very long string
		PagesDone:  0,
		TotalPages: 5,
		Options:    OCROptions{ProcessMode: "image"},
	}

	jobStore.addJob(job)
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Job added: id=test-job-123 doc_id=42 status=pending pages=0/5 mode=image")
	assert.NotContains(t, logOutput, "Long OCR Result")

	buf.Reset()
	jobStore.updatePagesDone(job.ID, 3)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Job pages done updated: id=test-job-123 doc_id=42 pages=3/5")

	buf.Reset()
	jobStore.updateJobStatus(job.ID, "completed", job.Result)
	logOutput = buf.String()
	assert.Contains(t, logOutput, "Job status updated: id=test-job-123 doc_id=42 status=completed pages=3/5 result_len=16000")
	assert.NotContains(t, logOutput, "Long OCR Result")
}
