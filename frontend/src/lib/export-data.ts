export type ExportFormat = 'xlsx' | 'xls' | 'csv' | 'txt';

export type ExportColumn<T> = {
  id?: string;
  header: string;
  value: (item: T, index: number) => unknown;
};

export function exportRows<T>(
  rows: T[],
  columns: ExportColumn<T>[],
  format: ExportFormat,
  fileName: string
) {
  const matrix = [
    columns.map(column => column.header),
    ...rows.map((row, rowIndex) => columns.map(column => column.value(row, rowIndex))),
  ];
  const filename = `${sanitizeExportFileName(fileName || `export-${localTimestamp()}`)}.${format}`;
  const blob = createExportBlob(matrix, format);
  downloadBlob(blob, filename);
}

export function localTimestamp() {
  const date = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}${pad(date.getMonth() + 1)}${pad(date.getDate())}${pad(date.getHours())}${pad(date.getMinutes())}`;
}

export function sanitizeExportFileName(value: string) {
  const printable = Array.from(value)
    .filter(char => (char.codePointAt(0) ?? 0) >= 32)
    .join('');
  const cleaned = printable
    .trim()
    .replace(/[<>:"/\\|?*]+/g, '-')
    .replace(/\s+/g, '-');
  return cleaned || `export-${localTimestamp()}`;
}

export function createExportBlob(matrix: unknown[][], format: ExportFormat) {
  if (format === 'csv') {
    return new Blob([toDelimited(matrix, ',')], { type: 'text/csv;charset=utf-8' });
  }
  if (format === 'txt') {
    return new Blob([toFixedWidthTable(matrix)], { type: 'text/plain;charset=utf-8' });
  }
  if (format === 'xls') {
    return new Blob([toExcelHtml(matrix)], { type: 'application/vnd.ms-excel;charset=utf-8' });
  }
  return new Blob([createXlsx(matrix)], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
}

function toDelimited(matrix: unknown[][], delimiter: string) {
  return (
    '\ufeff' +
    matrix.map(row => row.map(value => quote(value, delimiter)).join(delimiter)).join('\r\n')
  );
}

function toFixedWidthTable(matrix: unknown[][]) {
  const rows = matrix.map(row => row.map(safeText));
  if (rows.length === 0) return '\ufeff';
  const widths = columnWidths(matrix).map(width => Math.ceil((width + 4) / 8) * 8);
  const starts = widths.reduce<number[]>(offsets => {
    offsets.push((offsets.at(-1) ?? 0) + (offsets.length === 0 ? 0 : widths[offsets.length - 1]));
    return offsets;
  }, []);
  return '\ufeff' + rows.map(row => alignTextRow(row, starts)).join('\r\n');
}

function toExcelHtml(matrix: unknown[][]) {
  const widths = columnWidths(matrix);
  const rows = matrix
    .map(
      (row, index) =>
        `<tr>${row
          .map(value => {
            if (index === 0) {
              return `<th>${escapeHtml(safeText(value))}</th>`;
            }
            const dateSerial = excelDateSerial(value);
            if (dateSerial !== null) {
              return `<td style="mso-number-format:'yyyy/m/d'">${dateSerial}</td>`;
            }
            const cell = parseExportCell(value);
            if (cell.kind === 'number') {
              return `<td style="mso-number-format:General">${cell.value}</td>`;
            }
            if (cell.kind === 'percent') {
              return `<td style="mso-number-format:'${cell.decimals ? '0.00' : '0'}%'">${cell.value}</td>`;
            }
            return `<td>${escapeHtml(cell.text)}</td>`;
          })
          .join('')}</tr>`
    )
    .join('');
  return `<!doctype html><html xmlns:o="urn:schemas-microsoft-com:office" xmlns:x="urn:schemas-microsoft-com:office:excel"><head><meta charset="utf-8"><style>table{border-collapse:collapse}col{mso-width-source:userset}td,th{border:1px solid #d9e2ec;padding:4px 8px;text-align:center;vertical-align:middle;background-color:transparent;font-family:"宋体",SimSun,serif;white-space:nowrap}th{font-weight:700;mso-number-format:"\\@"}td{mso-number-format:"\\@"}</style></head><body><table><colgroup>${widths.map(width => `<col style="width:${Math.round(width * 11 + 34)}px">`).join('')}</colgroup>${rows}</table></body></html>`;
}

function createXlsx(matrix: unknown[][]) {
  const columns = columnWidths(matrix)
    .map(
      (width, index) =>
        `<col min="${index + 1}" max="${index + 1}" width="${Math.min(Math.max(width + 2, 9), 80)}" customWidth="1"/>`
    )
    .join('');
  const sheetRows = matrix
    .map(
      (row, rowIndex) =>
        `<row r="${rowIndex + 1}">${row
          .map((value, columnIndex) => {
            const ref = cellRef(rowIndex, columnIndex);
            if (rowIndex === 0) {
              return `<c r="${ref}" t="inlineStr" s="2"><is><t>${escapeXml(safeText(value))}</t></is></c>`;
            }
            if (typeof value === 'string' && value.includes('\n')) {
              const escapedLines = escapeXml(value).replace(/\n/g, '&#10;');
              return `<c r="${ref}" t="inlineStr" s="6"><is><t xml:space="preserve">${escapedLines}</t></is></c>`;
            }
            const dateSerial = excelDateSerial(value);
            if (dateSerial !== null) {
              return `<c r="${ref}" s="7"><v>${dateSerial}</v></c>`;
            }
            const cell = parseExportCell(value);
            if (cell.kind === 'number') {
              return `<c r="${ref}" s="3"><v>${cell.value}</v></c>`;
            }
            if (cell.kind === 'percent') {
              return `<c r="${ref}" s="${cell.decimals ? 5 : 4}"><v>${cell.value}</v></c>`;
            }
            return `<c r="${ref}" t="inlineStr" s="1"><is><t>${escapeXml(cell.text)}</t></is></c>`;
          })
          .join('')}</row>`
    )
    .join('');
  const files = new Map<string, string>([
    [
      '[Content_Types].xml',
      '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>',
    ],
    [
      '_rels/.rels',
      '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>',
    ],
    [
      'xl/workbook.xml',
      '<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>',
    ],
    [
      'xl/_rels/workbook.xml.rels',
      '<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>',
    ],
    [
      'xl/styles.xml',
      '<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="yyyy/m/d"/></numFmts><fonts count="2"><font><sz val="11"/><name val="宋体"/><family val="3"/><charset val="134"/></font><font><b/><sz val="11"/><name val="宋体"/><family val="3"/><charset val="134"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="2"><border><left/><right/><top/><bottom/></border><border><left style="thin"><color rgb="FFD9E2EC"/></left><right style="thin"><color rgb="FFD9E2EC"/></right><top style="thin"><color rgb="FFD9E2EC"/></top><bottom style="thin"><color rgb="FFD9E2EC"/></bottom></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="8"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="49" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="49" fontId="1" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyFont="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="0" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="9" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="10" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="49" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf><xf numFmtId="164" fontId="0" fillId="0" borderId="1" xfId="0" applyAlignment="1" applyBorder="1" applyNumberFormat="1"><alignment horizontal="center" vertical="center"/></xf></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>',
    ],
    [
      'xl/worksheets/sheet1.xml',
      `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols>${columns}</cols><sheetData>${sheetRows}</sheetData></worksheet>`,
    ],
  ]);
  return zipFiles(files);
}

function safeText(value: unknown) {
  const text = String(value ?? '')
    .replace(/\s+/g, ' ')
    .trim();
  return /^[=+\-@]/.test(text) && text.length > 1 ? `'${text}` : text;
}

type ExportCellValue =
  | { kind: 'text'; text: string }
  | { kind: 'number'; value: number }
  | { kind: 'percent'; value: number; decimals: boolean };

const excelDatePattern = /^\d{4}-\d{2}-\d{2}$/;

/* "YYYY-MM-DD" 文本转 Excel 日期序列号(自 1899-12-30 起算)，非日期文本返回 null */
function excelDateSerial(value: unknown) {
  if (typeof value !== 'string') return null;
  const text = value.trim();
  if (!excelDatePattern.test(text)) return null;
  const [year, month, day] = text.split('-').map(Number);
  const serial = Math.round((Date.UTC(year, month - 1, day) - Date.UTC(1899, 11, 30)) / 86400000);
  return serial > 0 ? serial : null;
}

/* 纯数字/纯小数/纯百分比（含数字型单元格值）转成真正的数值，避免 Excel “数字是文本类型”告警；
   前导零与超长数字串（如序列号）保持文本，防止丢失精度或前导 0 */
const numericPattern = /^-?(0|[1-9]\d*)(\.\d+)?$/;
const percentPattern = /^-?(0|[1-9]\d*)(\.\d+)?\s*%$/;
const maxNumericLength = 15;

function parseExportCell(value: unknown): ExportCellValue {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? { kind: 'number', value } : { kind: 'text', text: '' };
  }
  if (typeof value === 'string') {
    const text = value.replace(/\s+/g, ' ').trim();
    if (text.length > 0 && text.length <= maxNumericLength) {
      if (numericPattern.test(text)) {
        return { kind: 'number', value: Number(text) };
      }
      const percent = percentPattern.exec(text);
      if (percent) {
        return {
          kind: 'percent',
          value: Number(text.replace('%', '').trim()) / 100,
          decimals: text.includes('.'),
        };
      }
    }
  }
  return { kind: 'text', text: safeText(value) };
}

function quote(value: unknown, delimiter: string) {
  const text = safeText(value);
  return !text.includes(delimiter) && !/["\r\n]/.test(text)
    ? text
    : `"${text.replaceAll('"', '""')}"`;
}

function escapeXml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escapeHtml(value: string) {
  return escapeXml(value).replace(/'/g, '&#39;');
}

function cellRef(rowIndex: number, columnIndex: number) {
  let column = '';
  for (let value = columnIndex + 1; value > 0; value = Math.floor((value - 1) / 26)) {
    column = String.fromCharCode(65 + ((value - 1) % 26)) + column;
  }
  return `${column}${rowIndex + 1}`;
}

function displayWidth(value: string) {
  return Array.from(value).reduce(
    (width, char) => width + ((char.codePointAt(0) ?? 0) > 0xff ? 2 : 1),
    0
  );
}

function columnWidths(matrix: unknown[][]) {
  const columnCount = matrix.reduce((max, row) => Math.max(max, row.length), 0);
  return Array.from({ length: columnCount }, (_, index) =>
    Math.min(72, Math.max(6, ...matrix.map(row => displayWidth(safeText(row[index] ?? '')) + 2)))
  );
}

function alignTextRow(row: string[], starts: number[]) {
  let output = '';
  let position = 0;
  for (let index = 0; index < row.length; index += 1) {
    const target = starts[index] ?? position;
    if (position < target) output += ' '.repeat(target - position);
    const value = row[index] ?? '';
    output += value;
    position = target + displayWidth(value);
  }
  return output.trimEnd();
}

export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function zipFiles(files: Map<string, string>) {
  const encoder = new TextEncoder();
  const chunks: Uint8Array[] = [];
  const central: Uint8Array[] = [];
  let offset = 0;
  for (const [name, content] of files) {
    const nameBytes = encoder.encode(name);
    const data = encoder.encode(content);
    const checksum = crc32(data);
    const local = concat(
      u32(0x04034b50),
      u16(20),
      u16(0x800),
      u16(0),
      u16(0),
      u16(0),
      u32(checksum),
      u32(data.length),
      u32(data.length),
      u16(nameBytes.length),
      u16(0),
      nameBytes,
      data
    );
    chunks.push(local);
    central.push(
      concat(
        u32(0x02014b50),
        u16(20),
        u16(20),
        u16(0x800),
        u16(0),
        u16(0),
        u16(0),
        u32(checksum),
        u32(data.length),
        u32(data.length),
        u16(nameBytes.length),
        u16(0),
        u16(0),
        u16(0),
        u16(0),
        u32(0),
        u32(offset),
        nameBytes
      )
    );
    offset += local.length;
  }
  const centralSize = central.reduce((total, chunk) => total + chunk.length, 0);
  return concat(
    ...chunks,
    ...central,
    u32(0x06054b50),
    u16(0),
    u16(0),
    u16(files.size),
    u16(files.size),
    u32(centralSize),
    u32(offset),
    u16(0)
  );
}

function concat(...parts: Uint8Array[]) {
  const output = new Uint8Array(parts.reduce((total, part) => total + part.length, 0));
  let offset = 0;
  for (const part of parts) {
    output.set(part, offset);
    offset += part.length;
  }
  return output;
}

function u16(value: number) {
  const bytes = new Uint8Array(2);
  new DataView(bytes.buffer).setUint16(0, value, true);
  return bytes;
}
function u32(value: number) {
  const bytes = new Uint8Array(4);
  new DataView(bytes.buffer).setUint32(0, value >>> 0, true);
  return bytes;
}

const crcTable = new Uint32Array(256).map((_, index) => {
  let value = index;
  for (let bit = 0; bit < 8; bit += 1) value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1;
  return value >>> 0;
});

function crc32(data: Uint8Array) {
  let crc = 0xffffffff;
  for (const value of data) crc = crcTable[(crc ^ value) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}
