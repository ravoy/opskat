import type * as asset_entity from "@bindings/github.com/octays/opskat/internal/model/entity/asset_entity/models";
import { getSnippetRunner } from "@/lib/snippetRunners";
import { resolveSnippetContent } from "@/stores/snippetVariableStore";

/**
 * Whether a snippet category can be run, given the current category registry.
 * A category is runnable when its bound asset type has a registered runner.
 */
export function isRunnableCategoryId(categoryId: string, categories: { id: string; assetType: string }[]): boolean {
  const c = categories.find((x) => x.id === categoryId);
  return !!c && getSnippetRunner(c.assetType) !== undefined;
}

/**
 * Open the right tab for an asset and land the snippet content in its editor.
 * Never auto-executes. 含 {{变量}} 的片段先弹填写表单；取消则不打开任何 tab。
 */
export async function runSnippetOnAsset(asset: asset_entity.Asset, content: string): Promise<void> {
  const resolved = await resolveSnippetContent(content);
  if (resolved === null) return;
  const runner = getSnippetRunner(asset.Type);
  if (!runner) throw new Error(`snippetRun: unsupported asset type ${asset.Type}`);
  await runner(asset, resolved);
}
