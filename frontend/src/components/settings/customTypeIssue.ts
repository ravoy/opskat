import type { TFunction } from "i18next";
import type * as custom_type_entity from "@bindings/github.com/octays/opskat/internal/model/entity/custom_type_entity/models";

/** 后端校验问题只带问题码 + 参数(custom_type_entity.Issue),文案在语言包的
 * customType.issue.<code>(模板问题为 customType.issue.template.<code>),按界面语言翻译。 */
export function issueText(t: TFunction, issue: custom_type_entity.Issue): string {
  return t(`customType.issue.${issue.code}`, issue.params);
}
