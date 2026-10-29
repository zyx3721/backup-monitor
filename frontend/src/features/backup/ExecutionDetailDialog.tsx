import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ExecutionPair, ExecutionWideRow } from './ExecutionDetailRows';
import { ExecutionStateTag, MethodTag, formatTime } from './components';
import { executionDuration, formatExecutionID, sizeLabel } from './execution-utils';
import type { BackupExecution, BackupTask } from './types';

export function ExecutionDetailDialog({
  execution,
  task,
  onOpenChange,
}: {
  execution: BackupExecution | null;
  task?: BackupTask | null;
  onOpenChange: (open: boolean) => void;
}) {
  const veeamExecution =
    execution?.taskType === 'veeam' || (!execution?.taskType && task?.type === 'veeam');
  const title = execution?.taskName ?? task?.name ?? '';

  return (
    <Dialog open={Boolean(execution)} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[86vh] max-w-[1080px] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{title ? `执行详情 - ${title}` : '执行详情'}</DialogTitle>
        </DialogHeader>
        {execution ? (
          <div className="overflow-hidden rounded-lg border border-[var(--bm-border)] text-sm">
            <ExecutionPair
              label="主机"
              value={execution.hostName || task?.hostName || '-'}
              rightLabel="任务"
              rightValue={execution.taskName || task?.name || '-'}
            />
            <ExecutionPair
              label="备份方式"
              value={
                execution.taskType ? (
                  <MethodTag method={execution.taskType} />
                ) : task ? (
                  <MethodTag method={task.type} />
                ) : (
                  '-'
                )
              }
              rightLabel="状态"
              rightValue={<ExecutionStateTag state={execution.state} />}
            />
            <ExecutionPair
              label="触发"
              value={
                execution.trigger === 'manual'
                  ? '手动'
                  : execution.trigger === 'scheduled'
                    ? '定时'
                    : '-'
              }
              rightLabel="耗时"
              rightValue={executionDuration(execution)}
            />
            <ExecutionPair
              label="创建时间"
              value={formatTime(execution.creationTime)}
              rightLabel="执行时间"
              rightValue={formatTime(execution.startTime)}
            />
            <ExecutionPair
              label="结束时间"
              value={formatTime(execution.endTime)}
              rightLabel={veeamExecution ? '处理大小' : '备份大小'}
              rightValue={
                veeamExecution
                  ? sizeLabel(execution.processedSize)
                  : sizeLabel(execution.transferredSize || execution.processedSize)
              }
            />
            {veeamExecution ? (
              <ExecutionPair
                label="备份大小"
                value={sizeLabel(execution.transferredSize)}
                rightLabel="尝试次数"
                rightValue={
                  execution.attempt === undefined
                    ? '-'
                    : `${execution.attempt} / ${execution.maxAttempts ?? 1}`
                }
              />
            ) : null}
            {!veeamExecution && execution.artifactPath ? (
              <ExecutionWideRow label="备份产物" value={execution.artifactPath} />
            ) : null}
            <ExecutionPair
              label="最后尝试"
              value={
                execution.isLastAttempt === undefined ? '-' : execution.isLastAttempt ? '是' : '否'
              }
              rightLabel="会话 ID"
              rightValue={
                veeamExecution ? formatExecutionID(execution.sessionId) : execution.sessionId || '-'
              }
            />
            {veeamExecution ? (
              <>
                <ExecutionWideRow label="Job ID" value={formatExecutionID(execution.jobId)} />
                <ExecutionWideRow label="OIB ID" value={formatExecutionID(execution.oibId)} />
              </>
            ) : null}
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
