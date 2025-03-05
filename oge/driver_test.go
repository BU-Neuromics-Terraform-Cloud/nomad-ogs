package oge

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/stretchr/testify/require"
)

func TestOGEDriver_Fingerprint(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := testLogger()
	driver := NewOGEDriver(logger).(drivers.DriverPlugin)

	// Configure the driver
	err := driver.SetConfig(
		&base.Config{
			PluginConfig: []byte(`{"qsub_path": "echo", "qstat_path": "echo", "qdel_path": "echo"}`),
		},
	)
	require.NoError(t, err)

	// Get the fingerprint channel
	ch, err := driver.Fingerprint(ctx)
	require.NoError(t, err)

	// Get the first fingerprint result
	var fp *drivers.Fingerprint
	select {
	case fp = <-ch:
	case <-time.After(5 * time.Second):
		require.Fail(t, "timeout receiving fingerprint")
	}

	// Verify the driver is detected
	require.Equal(t, drivers.HealthStateHealthy, fp.Health)
}

func TestOGEDriver_StartTask_MinimalConfig(t *testing.T) {
	// This test would require a real OGE environment
	// For now, we'll just skip it
	t.Skip("Requires OGE environment")
}

func TestOGEDriver_TaskConfigSchema(t *testing.T) {
	t.Parallel()

	logger := testLogger()
	driver := NewOGEDriver(logger).(drivers.DriverPlugin)

	// Get the task config schema
	schema, err := driver.TaskConfigSchema()
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Verify required fields
	require.Contains(t, schema.Fields, "script")
	require.True(t, schema.Fields["script"].Required)
}

func testLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		JSONFormat: true,
		Name:       "oge-test",
	})
} 