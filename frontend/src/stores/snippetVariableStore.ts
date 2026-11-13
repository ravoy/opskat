import { create } from "zustand";
import { parseSnippetVariables, substituteSnippetVariables, type SnippetVariable } from "@/lib/snippetVariables";

/**
 * 片段变量填写弹窗的全局宿主状态 + 会话内变量值记忆。
 *
 * 弹窗由 App 挂一次（每个窗口一份）；任何入口（插入弹层、按资产运行）通过
 * resolveSnippetContent 拿到最终内容，取消时得到 null。
 */
export type SnippetVariableOutcome = { values: Record<string, string> } | { verbatim: true };

interface SnippetVariableState {
  /** 待填写的请求；非空即弹窗打开。 */
  pending: { variables: SnippetVariable[]; resolve: (outcome: SnippetVariableOutcome | null) => void } | null;
  /** 同一会话记住的值（按变量名），作为下次预填。 */
  sessionValues: Record<string, string>;
  openPrompt: (variables: SnippetVariable[], resolve: (outcome: SnippetVariableOutcome | null) => void) => void;
  /** values=确认填写；verbatim=原样插入；null=取消。 */
  settle: (outcome: SnippetVariableOutcome | null) => void;
  clearSessionValues: () => void;
}

export const useSnippetVariableStore = create<SnippetVariableState>((set, get) => ({
  pending: null,
  sessionValues: {},
  openPrompt: (variables, resolve) => set({ pending: { variables, resolve } }),
  settle: (outcome) => {
    const pending = get().pending;
    if (!pending) return;
    set({ pending: null });
    pending.resolve(outcome);
  },
  clearSessionValues: () => set({ sessionValues: {} }),
}));

/**
 * 无变量立即返回原文；有变量弹表单，确认返回替换后的内容，「原样插入」返回
 * 原文，取消返回 null。会话内同名变量用记住的值预填，确认后记住本次填写。
 */
export async function resolveSnippetContent(content: string): Promise<string | null> {
  const variables = parseSnippetVariables(content);
  if (variables.length === 0) return content;
  const outcome = await new Promise<SnippetVariableOutcome | null>((resolve) => {
    useSnippetVariableStore.getState().openPrompt(variables, resolve);
  });
  if (outcome === null) return null;
  if ("verbatim" in outcome) return content;
  useSnippetVariableStore.setState((s) => {
    const remembered = { ...s.sessionValues };
    for (const v of variables) {
      const filled = outcome.values[v.name];
      // 留空且无默认值的变量保留原样占位，不值得记住"空"。
      if (filled !== undefined && filled !== "") remembered[v.name] = filled;
    }
    return { sessionValues: remembered };
  });
  return substituteSnippetVariables(content, outcome.values);
}
