import { useRef } from 'react';

const tokenPattern =
  /("(?:\\.|[^"\\])*")(\s*:)?|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|\b(true|false|null)\b|([{}[\],:])/g;

type Token = { className: string; text: string };

const variablePattern = /(\{\{[^}\n]*\}\}|\$\{[^}\n]*\})/;
const escapePattern = /(\\[\s\S])/;

const LINE_HEIGHT = 19.2;
const PAD_Y = 10;
const PAD_X = 12;

function pushToken(tokens: Token[], className: string, text: string, withEscapes = false) {
  if (!text) return;
  for (const part of text.split(variablePattern)) {
    if (!part) continue;
    if (part.startsWith('{{') || part.startsWith('${')) {
      tokens.push({ className: 'bm-code-var', text: part });
      continue;
    }
    if (withEscapes) {
      for (const piece of part.split(escapePattern)) {
        if (!piece) continue;
        if (piece.length === 2 && piece.startsWith('\\')) {
          tokens.push({ className: 'bm-code-esc', text: piece });
        } else {
          tokens.push({ className, text: piece });
        }
      }
      continue;
    }
    tokens.push({ className, text: part });
  }
}

function tokenize(value: string): Token[] {
  const tokens: Token[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  tokenPattern.lastIndex = 0;
  while ((match = tokenPattern.exec(value)) !== null) {
    if (match.index > lastIndex) {
      pushToken(tokens, '', value.slice(lastIndex, match.index));
    }
    const [raw, stringToken, colonPart, numberToken, literalToken, punctToken] = match;
    if (stringToken) {
      if (colonPart) {
        pushToken(tokens, 'bm-code-key', stringToken, true);
        tokens.push({ className: 'bm-code-punct', text: colonPart });
      } else {
        pushToken(tokens, 'bm-code-string', stringToken, true);
      }
    } else if (numberToken) {
      tokens.push({ className: 'bm-code-number', text: raw });
    } else if (literalToken) {
      tokens.push({ className: 'bm-code-literal', text: raw });
    } else if (punctToken) {
      tokens.push({ className: 'bm-code-punct', text: raw });
    }
    lastIndex = match.index + raw.length;
    if (raw.length === 0) break;
  }
  if (lastIndex < value.length) {
    pushToken(tokens, '', value.slice(lastIndex));
  }
  return tokens;
}

export function CodeEditor({
  value,
  onChange,
  rows = 4,
  placeholder,
  invalid,
  ariaLabel,
}: {
  value: string;
  onChange: (value: string) => void;
  rows?: number;
  placeholder?: string;
  invalid?: boolean;
  ariaLabel?: string;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const textRef = useRef<HTMLTextAreaElement>(null);
  const measureRef = useRef<HTMLSpanElement>(null);
  const lineCount = Math.max(value.split('\n').length, 1);
  const tokens = tokenize(value);

  const syncCaret = () => {
    const textarea = textRef.current;
    const scroll = scrollRef.current;
    if (!textarea || !scroll) return;
    const position = textarea.selectionStart ?? 0;
    const before = textarea.value.slice(0, position);
    const line = before.split('\n').length - 1;
    const caretTop = line * LINE_HEIGHT + PAD_Y;
    const viewTop = scroll.scrollTop;
    const viewBottom = viewTop + scroll.clientHeight;
    if (caretTop + LINE_HEIGHT > viewBottom - 4) {
      scroll.scrollTop = caretTop - scroll.clientHeight + LINE_HEIGHT * 2;
    } else if (caretTop < viewTop + PAD_Y) {
      scroll.scrollTop = Math.max(0, caretTop - PAD_Y);
    }
    if (!measureRef.current) return;
    const lineStart = before.lastIndexOf('\n') + 1;
    measureRef.current.textContent = before.slice(lineStart) || ' ';
    const caretLeft = measureRef.current.offsetWidth + PAD_X;
    const viewRight = scroll.scrollLeft + scroll.clientWidth;
    if (caretLeft > viewRight - 16) {
      scroll.scrollLeft = caretLeft - scroll.clientWidth + 48;
    } else if (caretLeft - 20 < scroll.scrollLeft) {
      scroll.scrollLeft = Math.max(0, caretLeft - PAD_X - 48);
    }
  };

  return (
    <div className={`bm-code-editor${invalid ? ' is-invalid' : ''}`}>
      <div
        ref={scrollRef}
        className="bm-code-scroll"
        style={{ height: rows * LINE_HEIGHT + PAD_Y * 2 }}
      >
        <div className="bm-code-body">
          <div className="bm-code-gutter" aria-hidden="true">
            {Array.from({ length: lineCount }, (_, index) => (
              <div key={index}>{index + 1}</div>
            ))}
          </div>
          <div className="bm-code-code">
            <pre aria-hidden="true" className="bm-code-hl">
              {value ? (
                tokens.map((token, index) => (
                  <span key={index} className={token.className}>
                    {token.text}
                  </span>
                ))
              ) : (
                <span className="bm-code-placeholder">{placeholder}</span>
              )}
              {'\n'}
            </pre>
            <textarea
              ref={textRef}
              value={value}
              onChange={event => {
                onChange(event.target.value);
                requestAnimationFrame(syncCaret);
              }}
              onSelect={syncCaret}
              onClick={syncCaret}
              onKeyUp={syncCaret}
              aria-label={ariaLabel}
              aria-invalid={invalid}
              spellCheck={false}
              autoCapitalize="off"
              autoCorrect="off"
              autoComplete="off"
              wrap="off"
              className="bm-code-ta"
            />
            <span ref={measureRef} className="bm-code-measure" aria-hidden="true" />
          </div>
        </div>
      </div>
    </div>
  );
}
