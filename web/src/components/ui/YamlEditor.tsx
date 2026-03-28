import { Component, createSignal, createEffect, onMount } from 'solid-js';

interface YamlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
  placeholder?: string;
}

/**
 * YamlEditor - Code editor with line numbers and YAML syntax awareness.
 * Uses a textarea with custom styling for a code-editor experience.
 */
export const YamlEditor: Component<YamlEditorProps> = (props) => {
  let textareaRef: HTMLTextAreaElement | undefined;
  let lineNumbersRef: HTMLDivElement | undefined;
  const [lineCount, setLineCount] = createSignal(1);

  const updateLineCount = (text: string) => {
    const count = text.split('\n').length;
    setLineCount(count);
  };

  onMount(() => {
    updateLineCount(props.value);
  });

  createEffect(() => {
    updateLineCount(props.value);
  });

  const handleInput = (e: InputEvent) => {
    const target = e.currentTarget as HTMLTextAreaElement;
    const value = target.value;
    updateLineCount(value);
    props.onChange?.(value);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (props.readOnly) return;

    const target = e.currentTarget as HTMLTextAreaElement;

    // Tab key inserts 2 spaces
    if (e.key === 'Tab') {
      e.preventDefault();
      const start = target.selectionStart;
      const end = target.selectionEnd;
      const value = target.value;

      if (e.shiftKey) {
        // Shift+Tab: remove indentation
        const lineStart = value.lastIndexOf('\n', start - 1) + 1;
        const line = value.substring(lineStart, end);
        if (line.startsWith('  ')) {
          target.value = value.substring(0, lineStart) + line.substring(2);
          target.selectionStart = Math.max(start - 2, lineStart);
          target.selectionEnd = Math.max(end - 2, lineStart);
          props.onChange?.(target.value);
          updateLineCount(target.value);
        }
      } else {
        target.value = value.substring(0, start) + '  ' + value.substring(end);
        target.selectionStart = target.selectionEnd = start + 2;
        props.onChange?.(target.value);
        updateLineCount(target.value);
      }
    }

    // Enter key: auto-indent
    if (e.key === 'Enter') {
      e.preventDefault();
      const start = target.selectionStart;
      const value = target.value;

      // Find current line's indentation
      const lineStart = value.lastIndexOf('\n', start - 1) + 1;
      const line = value.substring(lineStart, start);
      const indent = line.match(/^(\s*)/)?.[1] || '';

      // Add extra indent if line ends with ':'
      const trimmed = line.trimEnd();
      const extraIndent = trimmed.endsWith(':') ? '  ' : '';

      const insertion = '\n' + indent + extraIndent;
      target.value = value.substring(0, start) + insertion + value.substring(start);
      target.selectionStart = target.selectionEnd = start + insertion.length;
      props.onChange?.(target.value);
      updateLineCount(target.value);
    }
  };

  const handleScroll = () => {
    if (lineNumbersRef && textareaRef) {
      lineNumbersRef.scrollTop = textareaRef.scrollTop;
    }
  };

  return (
    <div
      class="flex rounded-lg overflow-hidden font-mono text-sm"
      style={`height: ${props.height || '400px'}; background: #0d1117; border: 1px solid var(--border-primary);`}
    >
      {/* Line numbers */}
      <div
        ref={lineNumbersRef}
        class="flex-shrink-0 overflow-hidden select-none text-right pr-3 pl-3 py-3"
        style="background: #0d1117; color: #484f58; border-right: 1px solid #21262d; min-width: 48px; line-height: 1.6;"
      >
        {Array.from({ length: lineCount() }, (_, i) => (
          <div>{i + 1}</div>
        ))}
      </div>

      {/* Editor textarea */}
      <textarea
        ref={textareaRef}
        value={props.value}
        onInput={handleInput}
        onKeyDown={handleKeyDown}
        onScroll={handleScroll}
        readOnly={props.readOnly}
        placeholder={props.placeholder || 'Enter YAML configuration...'}
        spellcheck={false}
        class="flex-1 resize-none outline-none p-3"
        style={`
          background: #0d1117;
          color: #c9d1d9;
          line-height: 1.6;
          tab-size: 2;
          caret-color: #58a6ff;
          font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, Consolas, monospace;
          font-size: 13px;
          ${props.readOnly ? 'opacity: 0.7; cursor: default;' : ''}
        `}
      />
    </div>
  );
};
