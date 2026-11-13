import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Braces, Eraser } from "lucide-react";
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
} from "@opskat/ui";
import { useSnippetVariableStore } from "@/stores/snippetVariableStore";

/**
 * 片段变量填写弹窗（App 全局挂载一份）。pending 非空即打开：每个变量一行输入，
 * 预填顺序为会话记忆值 → 默认值。「原样插入」服务合法包含 `{{...}}` 的片段
 * （比如本来就要粘贴的 Terraform/Helm 模板）——变量语法必须可以 opt-out。
 */
export function SnippetVariablesDialog() {
  const { t } = useTranslation();
  const pending = useSnippetVariableStore((s) => s.pending);
  const sessionValues = useSnippetVariableStore((s) => s.sessionValues);
  const settle = useSnippetVariableStore((s) => s.settle);
  const clearSessionValues = useSnippetVariableStore((s) => s.clearSessionValues);

  const variables = pending?.variables ?? [];
  const [values, setValues] = useState<Record<string, string>>({});

  // 每次新请求重置表单：会话记忆值优先，其次默认值。渲染期对比上次请求，替代
  // effect 里的同步 setState（同 OpsctlApprovalDialog 的 prevTimedId 模式）。
  const [prevRequest, setPrevRequest] = useState<typeof pending | null | undefined>(undefined);
  if (prevRequest !== pending) {
    setPrevRequest(pending);
    if (pending) {
      const initial: Record<string, string> = {};
      for (const v of pending.variables) {
        initial[v.name] = sessionValues[v.name] ?? v.defaultValue ?? "";
      }
      setValues(initial);
    }
  }

  const confirm = () => {
    const filled: Record<string, string> = {};
    for (const v of variables) {
      const value = (values[v.name] ?? "").trim();
      if (value !== "") filled[v.name] = value;
    }
    settle({ values: filled });
  };

  return (
    <Dialog open={!!pending} onOpenChange={(open) => !open && settle(null)}>
      <DialogContent data-testid="snippet-variables-dialog" className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Braces className="h-4 w-4" />
            {t("snippet.variables.title")}
          </DialogTitle>
          <DialogDescription>{t("snippet.variables.description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3 max-h-[50vh] overflow-y-auto">
          {variables.map((v) => (
            <div key={v.name} className="space-y-1.5">
              <label htmlFor={`snippet-var-${v.name}`} className="text-xs font-mono text-muted-foreground">
                {v.name}
              </label>
              <Input
                id={`snippet-var-${v.name}`}
                data-testid={`snippet-var-${v.name}`}
                value={values[v.name] ?? ""}
                placeholder={v.defaultValue || t("snippet.variables.placeholder")}
                onChange={(e) => setValues((prev) => ({ ...prev, [v.name]: e.target.value }))}
                autoFocus={variables[0]?.name === v.name}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    confirm();
                  }
                }}
              />
            </div>
          ))}
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="ghost"
            data-testid="snippet-variables-clear-memory"
            className="mr-auto gap-1.5 text-muted-foreground"
            onClick={() => {
              clearSessionValues();
              const initial: Record<string, string> = {};
              for (const v of variables) initial[v.name] = v.defaultValue ?? "";
              setValues(initial);
            }}
          >
            <Eraser className="h-3.5 w-3.5" />
            {t("snippet.variables.clearMemory")}
          </Button>
          <Button variant="outline" data-testid="snippet-variables-verbatim" onClick={() => settle({ verbatim: true })}>
            {t("snippet.variables.insertVerbatim")}
          </Button>
          <Button data-testid="snippet-variables-confirm" onClick={confirm}>
            {t("snippet.variables.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
