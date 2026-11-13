import { describe, it, expect } from "vitest";
import { parseSnippetVariables, substituteSnippetVariables } from "./snippetVariables";

describe("parseSnippetVariables", () => {
  it("extracts variables in order of appearance, deduplicated", () => {
    const vars = parseSnippetVariables("kubectl logs {{pod}} -n {{ns}} --tail={{lines}} {{pod}}");
    expect(vars.map((v) => v.name)).toEqual(["pod", "ns", "lines"]);
  });

  it("reads defaults and distinguishes them from absent defaults", () => {
    const vars = parseSnippetVariables("psql -h {{host=localhost}} -p {{port}}");
    expect(vars[0]).toEqual({ name: "host", defaultValue: "localhost" });
    expect(vars[1]).toEqual({ name: "port", defaultValue: undefined });
  });

  it("content without variables yields an empty list", () => {
    expect(parseSnippetVariables("uptime")).toEqual([]);
    // Helm/Jinja/Go 模板语法的信号（. / $ 开头）不视为变量：粘贴模板不该弹表单。
    expect(parseSnippetVariables("helm template {{ .Values.name }} {{ $x }}")).toEqual([]);
  });

  it("trimmed names and defaults tolerate inner spaces", () => {
    const vars = parseSnippetVariables("echo {{ msg = hello world=x }} and {{ pod }}");
    expect(vars[0]).toEqual({ name: "msg", defaultValue: "hello world=x" });
    expect(vars[1]).toEqual({ name: "pod", defaultValue: undefined });
  });
});

describe("substituteSnippetVariables", () => {
  it("substitutes filled values", () => {
    expect(substituteSnippetVariables("kubectl logs {{pod}}", { pod: "api-7d9" })).toBe("kubectl logs api-7d9");
  });

  it("falls back to the default when left empty", () => {
    expect(substituteSnippetVariables("psql -h {{host=localhost}}", { host: "" })).toBe("psql -h localhost");
    expect(substituteSnippetVariables("psql -h {{host=localhost}}", {})).toBe("psql -h localhost");
  });

  it("keeps the placeholder when no value and no default exist", () => {
    expect(substituteSnippetVariables("rm {{path}}", {})).toBe("rm {{path}}");
  });

  it("replaces every occurrence of the same variable", () => {
    expect(substituteSnippetVariables("cp {{dir}}/a {{dir}}/b", { dir: "/srv" })).toBe("cp /srv/a /srv/b");
  });
});
