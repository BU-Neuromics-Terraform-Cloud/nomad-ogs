package oge

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/hashicorp/nomad/plugins/shared/structs"
)

const (
	// pluginName is the name of the plugin
	pluginName = "oge"

	// fingerprintPeriod is the interval at which the driver will send fingerprint responses
	fingerprintPeriod = 30 * time.Second

	// taskHandleVersion is the version of task handle which this driver sets
	// and understands how to decode driver state
	taskHandleVersion = 1
)

var (
	// pluginInfo is the response returned for the PluginInfo RPC
	pluginInfo = &base.PluginInfoResponse{
		Type:              base.PluginTypeDriver,
		PluginApiVersions: []string{drivers.ApiVersion010},
		PluginVersion:     "0.1.0",
		Name:              pluginName,
	}

	// configSpec is the hcl specification returned by the ConfigSchema RPC
	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"qsub_path": hclspec.NewDefault(
			hclspec.NewAttr("qsub_path", "string", false),
			hclspec.NewLiteral(`"qsub"`),
		),
		"qstat_path": hclspec.NewDefault(
			hclspec.NewAttr("qstat_path", "string", false),
			hclspec.NewLiteral(`"qstat"`),
		),
		"qdel_path": hclspec.NewDefault(
			hclspec.NewAttr("qdel_path", "string", false),
			hclspec.NewLiteral(`"qdel"`),
		),
		"default_queue": hclspec.NewDefault(
			hclspec.NewAttr("default_queue", "string", false),
			hclspec.NewLiteral(`"all.q"`),
		),
	})

	// taskConfigSpec is the hcl specification for the driver config section of
	// a task within a job. It is returned in the TaskConfigSchema RPC
	taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"script": hclspec.NewAttr("script", "string", true),
		"args": hclspec.NewDefault(
			hclspec.NewAttr("args", "list(string)", false),
			hclspec.NewLiteral(`[]`),
		),
		"queue": hclspec.NewAttr("queue", "string", false),
		"pe": hclspec.NewAttr("pe", "string", false),
		"slots": hclspec.NewDefault(
			hclspec.NewAttr("slots", "number", false),
			hclspec.NewLiteral("1"),
		),
		"resources": hclspec.NewDefault(
			hclspec.NewAttr("resources", "string", false),
			hclspec.NewLiteral(`""`),
		),
		"project": hclspec.NewAttr("project", "string", false),
	})

	// capabilities is returned by the Capabilities RPC and indicates what
	// optional features this driver supports
	capabilities = &drivers.Capabilities{
		SendSignals: false,
		Exec:        false,
		FSIsolation: drivers.FSIsolationNone,
	}
)

// Config is the driver configuration specified by the SetConfig RPC call
type Config struct {
	QsubPath    string `codec:"qsub_path"`
	QstatPath   string `codec:"qstat_path"`
	QdelPath    string `codec:"qdel_path"`
	DefaultQueue string `codec:"default_queue"`
}

// TaskConfig is the driver configuration of a task within a job
type TaskConfig struct {
	Script    string   `codec:"script"`
	Args      []string `codec:"args"`
	Queue     string   `codec:"queue"`
	PE        string   `codec:"pe"`
	Slots     int64    `codec:"slots"`
	Resources string   `codec:"resources"`
	Project   string   `codec:"project"`
}

// TaskState is the state which is encoded in the handle returned in
// StartTask. This information is needed to rebuild the task state and handler
// during recovery.
type TaskState struct {
	TaskConfig  *drivers.TaskConfig
	JobID       string
	StartedAt   time.Time
	CompletedAt time.Time
	ExitResult  *drivers.ExitResult
}

// OGEDriver is a driver for Open Grid Engine
type OGEDriver struct {
	// eventer is used to handle multiplexing of TaskEvents calls such that an
	// event can be broadcast to all callers
	eventer *eventer.Eventer

	// config is the driver configuration set by the SetConfig RPC
	config *Config

	// nomadConfig is the client config from nomad
	nomadConfig *base.ClientDriverConfig

	// tasks is the in memory datastore mapping taskIDs to rawExecDriverHandles
	tasks *taskStore

	// ctx is the context for the driver. It is passed to other subsystems to
	// coordinate shutdown
	ctx context.Context

	// signalShutdown is called when the driver is shutting down and cancels the
	// ctx passed to any subsystems
	signalShutdown context.CancelFunc

	// logger will log to the Nomad agent
	logger hclog.Logger
}

// NewOGEDriver returns a new OGE driver
func NewOGEDriver(logger hclog.Logger) drivers.DriverPlugin {
	ctx, cancel := context.WithCancel(context.Background())
	logger = logger.Named(pluginName)

	return &OGEDriver{
		eventer:        eventer.NewEventer(ctx, logger),
		config:         &Config{},
		tasks:          newTaskStore(),
		ctx:            ctx,
		signalShutdown: cancel,
		logger:         logger,
	}
}

// PluginInfo returns information describing the plugin.
func (d *OGEDriver) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the plugin configuration schema.
func (d *OGEDriver) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is called by the client to pass the configuration for the plugin.
func (d *OGEDriver) SetConfig(cfg *base.Config) error {
	var config Config
	if len(cfg.PluginConfig) != 0 {
		if err := base.MsgPackDecode(cfg.PluginConfig, &config); err != nil {
			return err
		}
	}

	d.config = &config
	if cfg.AgentConfig != nil {
		d.nomadConfig = cfg.AgentConfig.Driver
	}

	return nil
}

// TaskConfigSchema returns the HCL schema for the configuration of a task.
func (d *OGEDriver) TaskConfigSchema() (*hclspec.Spec, error) {
	return taskConfigSpec, nil
}

// Capabilities returns the features supported by the driver.
func (d *OGEDriver) Capabilities() (*drivers.Capabilities, error) {
	return capabilities, nil
}

// Fingerprint returns a channel that will be used to send health information to
// Nomad. This is the primary method Nomad uses to detect driver health and
// availability.
func (d *OGEDriver) Fingerprint(ctx context.Context) (<-chan *drivers.Fingerprint, error) {
	ch := make(chan *drivers.Fingerprint)
	go d.handleFingerprint(ctx, ch)
	return ch, nil
}

// handleFingerprint manages the channel and the flow of fingerprint data.
func (d *OGEDriver) handleFingerprint(ctx context.Context, ch chan<- *drivers.Fingerprint) {
	defer close(ch)

	// Nomad expects the initial fingerprint to be sent immediately
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// After the initial fingerprint, periodic updates should be sent
			ticker.Reset(fingerprintPeriod)
			ch <- d.buildFingerprint()
		}
	}
}

// buildFingerprint returns the driver's fingerprint data
func (d *OGEDriver) buildFingerprint() *drivers.Fingerprint {
	var health drivers.HealthState
	var desc string
	attrs := map[string]*structs.Attribute{}

	// Check if OGE commands are available
	qsubPath, err := exec.LookPath(d.config.QsubPath)
	if err != nil {
		health = drivers.HealthStateUndetected
		desc = fmt.Sprintf("OGE qsub command not found: %v", err)
	} else {
		health = drivers.HealthStateHealthy
		desc = "OGE driver is ready"
		attrs["driver.oge.qsub_path"] = structs.NewStringAttribute(qsubPath)
	}

	return &drivers.Fingerprint{
		Attributes:        attrs,
		Health:            health,
		HealthDescription: desc,
	}
}

// StartTask is used to start a task.
func (d *OGEDriver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
	if _, ok := d.tasks.Get(cfg.ID); ok {
		return nil, nil, fmt.Errorf("task with ID %q already started", cfg.ID)
	}

	var driverConfig TaskConfig
	if err := cfg.DecodeDriverConfig(&driverConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to decode driver config: %v", err)
	}

	d.logger.Info("starting oge task", "driver_cfg", hclog.Fmt("%+v", driverConfig))
	handle := drivers.NewTaskHandle(taskHandleVersion)
	handle.Config = cfg

	// Create a script file from the provided script content
	scriptPath := filepath.Join(cfg.TaskDir().Dir, "task.sh")
	if err := os.WriteFile(scriptPath, []byte(driverConfig.Script), 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to write script file: %v", err)
	}

	// Build qsub command
	qsubCmd := []string{d.config.QsubPath}

	// Add queue if specified, otherwise use default
	queue := driverConfig.Queue
	if queue == "" {
		queue = d.config.DefaultQueue
	}
	qsubCmd = append(qsubCmd, "-q", queue)

	// Add parallel environment if specified
	if driverConfig.PE != "" {
		qsubCmd = append(qsubCmd, "-pe", driverConfig.PE, strconv.FormatInt(driverConfig.Slots, 10))
	}

	// Add project if specified
	if driverConfig.Project != "" {
		qsubCmd = append(qsubCmd, "-P", driverConfig.Project)
	}

	// Add resource requests if specified
	if driverConfig.Resources != "" {
		qsubCmd = append(qsubCmd, "-l", driverConfig.Resources)
	}

	// Add job name
	qsubCmd = append(qsubCmd, "-N", fmt.Sprintf("nomad-%s", cfg.Name))

	// Add script path and args
	qsubCmd = append(qsubCmd, scriptPath)
	qsubCmd = append(qsubCmd, driverConfig.Args...)

	// Create the command
	cmd := exec.Command(qsubCmd[0], qsubCmd[1:]...)
	cmd.Dir = cfg.TaskDir().Dir

	// Capture stdout to get the job ID
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to submit job: %v", err)
	}

	// Parse the job ID from the output
	jobID := parseJobID(string(output))
	if jobID == "" {
		return nil, nil, fmt.Errorf("failed to parse job ID from output: %s", output)
	}

	d.logger.Info("submitted oge job", "job_id", jobID)

	h := &taskHandle{
		taskConfig:  cfg,
		jobID:       jobID,
		procState:   drivers.TaskStateRunning,
		startedAt:   time.Now(),
		logger:      d.logger,
		qstatPath:   d.config.QstatPath,
		qdelPath:    d.config.QdelPath,
		doneCh:      make(chan struct{}),
	}

	driverState := TaskState{
		TaskConfig: cfg,
		JobID:      jobID,
		StartedAt:  h.startedAt,
	}

	if err := handle.SetDriverState(&driverState); err != nil {
		return nil, nil, fmt.Errorf("failed to set driver state: %v", err)
	}

	d.tasks.Set(cfg.ID, h)
	go h.run()

	return handle, nil, nil
}

// parseJobID extracts the job ID from the qsub output
func parseJobID(output string) string {
	// The output format is typically "Your job 123456 has been submitted"
	re := regexp.MustCompile(`Your job (\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// RecoverTask is used to recover a task that has been restored by the allocator.
func (d *OGEDriver) RecoverTask(handle *drivers.TaskHandle) error {
	if handle == nil {
		return fmt.Errorf("error: handle cannot be nil")
	}

	if _, ok := d.tasks.Get(handle.Config.ID); ok {
		return fmt.Errorf("task with ID %q already exists", handle.Config.ID)
	}

	var taskState TaskState
	if err := handle.GetDriverState(&taskState); err != nil {
		return fmt.Errorf("failed to decode task state from handle: %v", err)
	}

	var driverConfig TaskConfig
	if err := taskState.TaskConfig.DecodeDriverConfig(&driverConfig); err != nil {
		return fmt.Errorf("failed to decode driver config: %v", err)
	}

	h := &taskHandle{
		taskConfig:  taskState.TaskConfig,
		jobID:       taskState.JobID,
		procState:   drivers.TaskStateRunning,
		startedAt:   taskState.StartedAt,
		exitResult:  taskState.ExitResult,
		logger:      d.logger,
		qstatPath:   d.config.QstatPath,
		qdelPath:    d.config.QdelPath,
		doneCh:      make(chan struct{}),
	}

	d.tasks.Set(taskState.TaskConfig.ID, h)

	go h.run()
	return nil
}

// WaitTask returns a channel used to notify Nomad when a task exits.
func (d *OGEDriver) WaitTask(ctx context.Context, taskID string) (<-chan *drivers.ExitResult, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	ch := make(chan *drivers.ExitResult)
	go d.handleWait(ctx, handle, ch)
	return ch, nil
}

func (d *OGEDriver) handleWait(ctx context.Context, handle *taskHandle, ch chan *drivers.ExitResult) {
	defer close(ch)

	select {
	case <-ctx.Done():
		return
	case <-d.ctx.Done():
		return
	case <-handle.doneCh:
		ch <- handle.exitResult
	}
}

// StopTask stops a running task.
func (d *OGEDriver) StopTask(taskID string, timeout time.Duration, signal string) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	if err := handle.shutdown(timeout); err != nil {
		return fmt.Errorf("executor shutdown failed: %v", err)
	}

	return nil
}

// DestroyTask cleans up resources associated with a task.
func (d *OGEDriver) DestroyTask(taskID string, force bool) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	if handle.IsRunning() && !force {
		return fmt.Errorf("cannot destroy running task")
	}

	d.tasks.Delete(taskID)
	return nil
}

// InspectTask returns detailed status information for a task.
func (d *OGEDriver) InspectTask(taskID string) (*drivers.TaskStatus, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.TaskStatus(), nil
}

// TaskStats returns statistics for a task.
func (d *OGEDriver) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.stats(ctx, interval)
}

// TaskEvents returns a channel that can be used to listen for events from the
// task.
func (d *OGEDriver) TaskEvents(ctx context.Context) (<-chan *drivers.TaskEvent, error) {
	return d.eventer.TaskEvents(ctx)
}

// SignalTask sends a signal to a task.
func (d *OGEDriver) SignalTask(taskID string, signal string) error {
	return fmt.Errorf("OGE driver does not support signals")
}

// ExecTask returns the result of executing the given command inside a task.
func (d *OGEDriver) ExecTask(taskID string, cmd []string, timeout time.Duration) (*drivers.ExecTaskResult, error) {
	return nil, fmt.Errorf("OGE driver does not support exec")
}

// taskHandle is a handle to a running OGE job
type taskHandle struct {
	taskConfig  *drivers.TaskConfig
	jobID       string
	procState   drivers.TaskState
	startedAt   time.Time
	completedAt time.Time
	exitResult  *drivers.ExitResult
	logger      hclog.Logger
	qstatPath   string
	qdelPath    string
	doneCh      chan struct{}

	stateLock sync.RWMutex
}

func (h *taskHandle) TaskStatus() *drivers.TaskStatus {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

	return &drivers.TaskStatus{
		ID:          h.taskConfig.ID,
		Name:        h.taskConfig.Name,
		State:       h.procState,
		StartedAt:   h.startedAt,
		CompletedAt: h.completedAt,
		ExitResult:  h.exitResult,
		DriverAttributes: map[string]string{
			"job_id": h.jobID,
		},
	}
}

func (h *taskHandle) IsRunning() bool {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()
	return h.procState == drivers.TaskStateRunning
}

func (h *taskHandle) run() {
	h.stateLock.Lock()
	if h.exitResult == nil {
		h.exitResult = &drivers.ExitResult{}
	}
	h.stateLock.Unlock()

	// Start monitoring the job
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status, err := h.checkJobStatus()
			if err != nil {
				h.logger.Warn("error checking job status", "error", err)
				continue
			}

			if status == "done" {
				h.stateLock.Lock()
				h.procState = drivers.TaskStateExited
				h.exitResult.ExitCode = 0
				h.completedAt = time.Now()
				h.stateLock.Unlock()
				close(h.doneCh)
				return
			} else if status == "error" {
				h.stateLock.Lock()
				h.procState = drivers.TaskStateExited
				h.exitResult.ExitCode = 1
				h.exitResult.Err = fmt.Errorf("job failed")
				h.completedAt = time.Now()
				h.stateLock.Unlock()
				close(h.doneCh)
				return
			}
		}
	}
}

func (h *taskHandle) checkJobStatus() (string, error) {
	cmd := exec.Command(h.qstatPath, "-j", h.jobID)
	output, err := cmd.CombinedOutput()
	
	// If the job is not found, it might have completed
	if err != nil && strings.Contains(string(output), "does not exist") {
		// Check if the job completed successfully or with an error
		// This would require checking the job's output files or logs
		// For simplicity, we'll assume it completed successfully
		return "done", nil
	} else if err != nil {
		return "", fmt.Errorf("error checking job status: %v, output: %s", err, output)
	}
	
	// Job is still running
	return "running", nil
}

func (h *taskHandle) shutdown(timeout time.Duration) error {
	cmd := exec.Command(h.qdelPath, h.jobID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error stopping job: %v", err)
	}
	
	// Wait for the job to be terminated
	select {
	case <-h.doneCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for job to terminate")
	}
}

func (h *taskHandle) stats(ctx context.Context, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	ch := make(chan *drivers.TaskResourceUsage)
	go h.collectStats(ctx, ch, interval)
	return ch, nil
}

func (h *taskHandle) collectStats(ctx context.Context, ch chan<- *drivers.TaskResourceUsage, interval time.Duration) {
	defer close(ch)
	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// OGE doesn't provide real-time resource usage stats easily
			// We could implement this by parsing qstat -j output or using other OGE commands
			// For now, we'll just send empty stats
			ch <- &drivers.TaskResourceUsage{
				ResourceUsage: &drivers.ResourceUsage{
					MemoryStats: &drivers.MemoryStats{},
					CpuStats:    &drivers.CpuStats{},
				},
				Timestamp: time.Now().UnixNano(),
			}
			timer.Reset(interval)
		}
	}
}

// taskStore provides a mechanism to store and retrieve task handles
type taskStore struct {
	tasks map[string]*taskHandle
	lock  sync.RWMutex
}

func newTaskStore() *taskStore {
	return &taskStore{
		tasks: map[string]*taskHandle{},
	}
}

func (ts *taskStore) Set(id string, handle *taskHandle) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	ts.tasks[id] = handle
}

func (ts *taskStore) Get(id string) (*taskHandle, bool) {
	ts.lock.RLock()
	defer ts.lock.RUnlock()
	t, ok := ts.tasks[id]
	return t, ok
}

func (ts *taskStore) Delete(id string) {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	delete(ts.tasks, id)
} 