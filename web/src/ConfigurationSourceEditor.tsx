import { json, jsonParseLinter } from "@codemirror/lang-json";
import { linter } from "@codemirror/lint";
import CodeMirror from "@uiw/react-codemirror";

interface ConfigurationSourceEditorProps {
  value: string;
  editing: boolean;
  loading: boolean;
  onChange: (value: string) => void;
}

export default function ConfigurationSourceEditor({ value, editing, loading, onChange }: ConfigurationSourceEditorProps) {
  return (
    <CodeMirror
      value={value}
      height="520px"
      extensions={[json(), linter(jsonParseLinter())]}
      editable={editing && !loading}
      readOnly={!editing || loading}
      basicSetup={{ foldGutter: true, lineNumbers: true, highlightActiveLine: editing, bracketMatching: true }}
      onChange={onChange}
      aria-label="配置源码 JSON"
    />
  );
}
