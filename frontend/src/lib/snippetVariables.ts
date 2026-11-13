// 代码片段的变量语法：`{{name}}` / `{{name=default}}`。与自定义类型凭据模板的
// `{{apiKey}}` 同一心智模型。变量只活在片段文本里——不需要数据模型迁移；不含
// 变量的片段走原路径，零打扰。

export interface SnippetVariable {
  name: string;
  /** `{{name=default}}` 的默认值；`{{name}}` 为 undefined。 */
  defaultValue?: string;
}

// 名称段：不以 `.` / `$` / 空白开头（那些是 Helm/Jinja/Go 模板语法的信号，不应弹
// 填写表单——粘贴模板的场景由弹窗的「原样插入」兜底）；默认值段可含 `=` 与空格。
// 两侧的空白都被吃掉，名称与默认值均 trim。
const VARIABLE_RE = /\{\{\s*([^{}=$.\s][^{}=]*?)\s*(?:=\s*([^{}]*?)\s*)?\}\}/g;

/**
 * 按出现顺序提取片段里的变量（去重，首个默认值生效）。
 */
export function parseSnippetVariables(content: string): SnippetVariable[] {
  const seen = new Map<string, SnippetVariable>();
  for (const m of content.matchAll(VARIABLE_RE)) {
    const name = m[1].trim();
    if (!name) continue;
    if (!seen.has(name)) {
      seen.set(name, { name, defaultValue: m[2] });
    }
  }
  return [...seen.values()];
}

/**
 * 用填写值替换全部变量。未填写的变量保留默认值；既无值也无默认值的保留原样
 * `{{name}}`——插入后仍能看出缺了什么，而不是悄悄吞掉占位符。
 */
export function substituteSnippetVariables(content: string, values: Record<string, string>): string {
  return content.replace(VARIABLE_RE, (whole, rawName: string, defaultValue?: string) => {
    const name = rawName.trim();
    if (!name) return whole;
    const value = values[name];
    if (value !== undefined && value !== "") return value;
    if (defaultValue !== undefined) return defaultValue;
    return whole;
  });
}
