import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SnippetVariablesDialog } from "../SnippetVariablesDialog";
import { resolveSnippetContent, useSnippetVariableStore } from "@/stores/snippetVariableStore";

// 变量填写弹窗的三个结局 + 会话记忆：确认（替换并记住）、原样插入（不替换）、
// 取消（null）。预填顺序：会话记忆值 → 默认值。

describe("SnippetVariablesDialog", () => {
  beforeEach(() => {
    useSnippetVariableStore.setState({ pending: null, sessionValues: {} });
  });

  it("content without variables passes through untouched", async () => {
    await expect(resolveSnippetContent("uptime -p")).resolves.toBe("uptime -p");
  });

  it("confirm substitutes values and remembers them for the session", async () => {
    render(<SnippetVariablesDialog />);

    const promise = resolveSnippetContent("kubectl logs {{pod}} -n {{ns}}");
    // 默认值预填（ns=prod），pod 无默认为空。
    await screen.findByTestId("snippet-variables-dialog");
    fireEvent.change(screen.getByTestId("snippet-var-pod"), { target: { value: "api-7d9" } });
    fireEvent.change(screen.getByTestId("snippet-var-ns"), { target: { value: "kube-system" } });
    fireEvent.click(screen.getByTestId("snippet-variables-confirm"));

    await expect(promise).resolves.toBe("kubectl logs api-7d9 -n kube-system");
    // 会话记忆：同名变量下次预填记住的值而非默认值。
    const second = resolveSnippetContent("kubectl logs {{pod}} -n {{ns=prod}}");
    await screen.findByTestId("snippet-variables-dialog");
    expect((screen.getByTestId("snippet-var-pod") as HTMLInputElement).value).toBe("api-7d9");
    expect((screen.getByTestId("snippet-var-ns") as HTMLInputElement).value).toBe("kube-system");
    fireEvent.click(screen.getByTestId("snippet-variables-confirm"));
    await expect(second).resolves.toBe("kubectl logs api-7d9 -n kube-system");
  });

  it("empty fields fall back to defaults; no default keeps the placeholder", async () => {
    render(<SnippetVariablesDialog />);

    const promise = resolveSnippetContent("psql -h {{host=localhost}} -p {{port}}");
    await screen.findByTestId("snippet-variables-dialog");
    fireEvent.click(screen.getByTestId("snippet-variables-confirm"));

    await expect(promise).resolves.toBe("psql -h localhost -p {{port}}");
  });

  it("insert verbatim returns the original content untouched", async () => {
    render(<SnippetVariablesDialog />);

    const promise = resolveSnippetContent("helm template {{ .release }} {{name}}");
    await screen.findByTestId("snippet-variables-dialog");
    fireEvent.click(screen.getByTestId("snippet-variables-verbatim"));

    await expect(promise).resolves.toBe("helm template {{ .release }} {{name}}");
  });

  it("cancel resolves to null and nothing is remembered", async () => {
    render(<SnippetVariablesDialog />);

    const promise = resolveSnippetContent("rm {{path}}");
    await screen.findByTestId("snippet-variables-dialog");
    fireEvent.change(screen.getByTestId("snippet-var-path"), { target: { value: "/tmp/x" } });
    fireEvent.keyDown(screen.getByTestId("snippet-var-path"), { key: "Escape" });

    await expect(promise).resolves.toBeNull();
    expect(useSnippetVariableStore.getState().sessionValues).toEqual({});
  });

  it("clear-memory wipes remembered values and reprefills defaults", async () => {
    useSnippetVariableStore.setState({ sessionValues: { host: "remembered.example" } });
    render(<SnippetVariablesDialog />);

    const promise = resolveSnippetContent("psql -h {{host=localhost}}");
    await screen.findByTestId("snippet-variables-dialog");
    expect((screen.getByTestId("snippet-var-host") as HTMLInputElement).value).toBe("remembered.example");
    fireEvent.click(screen.getByTestId("snippet-variables-clear-memory"));
    expect((screen.getByTestId("snippet-var-host") as HTMLInputElement).value).toBe("localhost");
    fireEvent.click(screen.getByTestId("snippet-variables-confirm"));

    await expect(promise).resolves.toBe("psql -h localhost");
    expect(useSnippetVariableStore.getState().sessionValues).toEqual({ host: "localhost" });
  });
});
