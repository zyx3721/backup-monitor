//go:build windows

package discover

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const scheduledTaskScript = `
$ErrorActionPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$items = @()
Get-ScheduledTask | ForEach-Object {
  $task = $_
  $info = $null
  try { $info = Get-ScheduledTaskInfo -TaskName $task.TaskName -TaskPath $task.TaskPath -ErrorAction Stop } catch { }
  $trigger = $null
  foreach ($candidate in $task.Triggers) {
    if ($null -eq $trigger) { $trigger = $candidate }
    if ($candidate.Enabled) { $trigger = $candidate; break }
  }
  $action = $task.Actions | Select-Object -First 1
  $execute = ''
  $arguments = ''
  if ($null -ne $action) {
    $execute = [string]$action.Execute
    $arguments = [string]$action.Arguments
  }
  $daysOfWeek = ''
  $daysOfMonth = ''
  $interval = ''
  $schedule = ''
  $triggerType = ''
  $enabled = $false
  if ($null -ne $trigger) {
    if ($null -ne $trigger.DaysOfWeek) { $daysOfWeek = [string]$trigger.DaysOfWeek }
    if ($null -ne $trigger.DaysOfMonth) { $daysOfMonth = ($trigger.DaysOfMonth -join ',') }
    if ($null -ne $trigger.Repetition -and $null -ne $trigger.Repetition.Interval) { $interval = [string]$trigger.Repetition.Interval }
    if ($null -ne $trigger.StartBoundary) { $schedule = [string]$trigger.StartBoundary }
    $triggerType = [string]$trigger.CimClass.CimClassName
    $enabled = [bool]$trigger.Enabled
  }
  $lastRun = ''
  $lastResult = 0
  if ($null -ne $info) {
    if ($info.LastRunTime) { $lastRun = $info.LastRunTime.ToString('o') }
    $lastResult = [int64]$info.LastTaskResult
  }
  $items += [pscustomobject]@{
    TaskName = [string]$task.TaskName
    TaskPath = [string]$task.TaskPath
    State = [string]$task.State
    LastRunTime = $lastRun
    LastTaskResult = $lastResult
    Execute = $execute
    Arguments = $arguments
    Comment = [string]$task.Description
    Schedule = $schedule
    TriggerType = $triggerType
    DaysOfWeek = $daysOfWeek
    DaysOfMonth = $daysOfMonth
    RepetitionInterval = $interval
    Enabled = $enabled
  }
}
$items | ConvertTo-Json -Depth 4 -Compress
`

// List 采集 Windows 计划任务，依次回退 PowerShell 与 schtasks CSV。
func List(ctx context.Context, filter Filter) ([]Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	output, powerShellErr := runPowerShell(timeoutCtx)
	if powerShellErr == nil {
		tasks, err := ParseScheduledTasksJSON(output, filter)
		if err != nil {
			powerShellErr = fmt.Errorf("parse PowerShell output: %w", err)
		} else if len(tasks) > 0 {
			return tasks, nil
		}
	}

	csvOutput, csvErr := runSchtasks(timeoutCtx)
	if csvErr == nil {
		tasks, err := ParseSchtasksCSV(csvOutput, filter)
		if err != nil {
			csvErr = fmt.Errorf("parse schtasks output: %w", err)
		} else if len(tasks) > 0 {
			return tasks, nil
		}
	}

	return nil, errors.Join(
		stageError("query scheduled tasks with PowerShell", powerShellErr),
		stageError("query scheduled tasks with schtasks", csvErr),
	)
}

// ApplyCronByKey 在当前平台上暂不支持修改 Windows 计划任务。
func ApplyCronByKey(ctx context.Context, key, expression string) (string, error) {
	return "Windows 计划任务暂不支持修改执行计划", fmt.Errorf("set_cron is not supported on Windows")
}

func runPowerShell(ctx context.Context) ([]byte, error) {
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", scheduledTaskScript)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func runSchtasks(ctx context.Context) ([]byte, error) {
	command := exec.CommandContext(ctx, "cmd.exe", "/c", "chcp 65001>nul & schtasks /query /fo CSV /v")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func stageError(stage string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", stage, err)
}
