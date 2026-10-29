import { useEffect, useMemo, useState } from 'react';
import { Check, Download, FileDown } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  exportRows,
  localTimestamp,
  sanitizeExportFileName,
  type ExportColumn,
  type ExportFormat,
} from '@/lib/export-data';
import { recordExportAudit } from '@/lib/api';

const formats: Array<{ value: ExportFormat; label: string }> = [
  { value: 'xlsx', label: 'XLSX' },
  { value: 'xls', label: 'XLS' },
  { value: 'csv', label: 'CSV' },
  { value: 'txt', label: 'TXT' },
];

export function ExportDialog<T>({
  open,
  onOpenChange,
  title,
  description,
  filenamePrefix,
  items,
  columns,
  successText,
  auditModule,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  filenamePrefix: string;
  items: T[];
  columns: ExportColumn<T>[];
  successText: string;
  auditModule: string;
}) {
  const [format, setFormat] = useState<ExportFormat>('xlsx');
  const [filename, setFilename] = useState(`${filenamePrefix}-${localTimestamp()}`);
  const [selectedColumnIds, setSelectedColumnIds] = useState<string[]>(() =>
    columns.map(column => column.id ?? column.header)
  );

  useEffect(() => {
    if (!open) return;
    setFormat('xlsx');
    setFilename(`${filenamePrefix}-${localTimestamp()}`);
    setSelectedColumnIds(columns.map(column => column.id ?? column.header));
  }, [columns, filenamePrefix, open]);

  const selectedColumns = useMemo(
    () => columns.filter(column => selectedColumnIds.includes(column.id ?? column.header)),
    [columns, selectedColumnIds]
  );
  const previewName = `${sanitizeExportFileName(filename || `${filenamePrefix}-${localTimestamp()}`)}.${format}`;
  const allSelected = selectedColumns.length === columns.length;

  function handleExport() {
    if (selectedColumns.length === 0) {
      toast.warning('请选择至少一个导出字段');
      return;
    }
    exportRows(items, selectedColumns, format, filename);
    toast.success(successText);
    onOpenChange(false);
    void recordExportAudit(auditModule, items.length).catch(() => undefined);
  }

  function toggleAllColumns() {
    setSelectedColumnIds(allSelected ? [] : columns.map(column => column.id ?? column.header));
  }

  function toggleColumn(id: string) {
    setSelectedColumnIds(current =>
      current.includes(id) ? current.filter(item => item !== id) : [...current, id]
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="bm-dialog-panel max-h-[88vh] gap-0 overflow-hidden p-0 sm:max-w-[680px]"
        onOpenAutoFocus={event => event.preventDefault()}
      >
        <DialogHeader className="border-b border-[var(--bm-border)] px-5 py-4">
          <DialogTitle>导出{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="bm-hidden-scrollbar space-y-5 overflow-y-auto p-5">
          <div className="grid gap-4 md:grid-cols-[1fr_180px]">
            <label className="space-y-1.5 text-xs text-[var(--bm-text-muted)]">
              <div>导出名称</div>
              <input
                value={filename}
                onChange={event => setFilename(event.target.value)}
                className="bm-form-control h-10 w-full rounded-lg px-3 text-sm"
              />
            </label>
            <div className="space-y-1.5 text-xs text-[var(--bm-text-muted)]">
              <div>扩展名</div>
              <Select value={format} onValueChange={value => setFormat(value as ExportFormat)}>
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {formats.map(item => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-3 rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)]/60 p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs font-semibold text-[var(--bm-text)]">导出字段</p>
                <p className="mt-1 text-xs text-[var(--bm-text-muted)]">
                  默认导出全部字段，可按需取消
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={toggleAllColumns}>
                {allSelected ? '取消全选' : '全选'}
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
              {columns.map(column => {
                const id = column.id ?? column.header;
                const selected = selectedColumnIds.includes(id);
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => toggleColumn(id)}
                    className={`bm-action-button bm-audit-field-button flex h-10 min-w-0 items-center gap-2 rounded-lg border px-3 text-left text-sm transition ${selected ? 'is-selected' : ''}`}
                  >
                    <span
                      className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${selected ? 'border-teal-400/70 bg-teal-400/20' : 'border-[var(--bm-border)]'}`}
                    >
                      {selected ? <Check size={12} /> : null}
                    </span>
                    <span className="min-w-0 truncate">{column.header}</span>
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] px-3 py-3 text-sm text-[var(--bm-text-muted)]">
            <FileDown size={16} />
            <span className="min-w-0 flex-1 truncate">{previewName}</span>
            <span className="shrink-0 text-xs">
              {selectedColumns.length}/{columns.length} 列
            </span>
          </div>
        </div>
        <DialogFooter className="border-t border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)]/40 px-5 py-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            onClick={handleExport}
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              color: 'var(--bm-accent-text)',
              background: 'rgba(59,130,246,0.1)',
            }}
          >
            <Download size={15} />
            导出
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
