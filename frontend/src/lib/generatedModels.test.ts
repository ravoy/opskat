import { expect, it } from "vitest";
import { Descriptor } from "@bindings/github.com/octays/opskat/pkg/dbdriver/models";

it("round-trips a generated model with omitted optional nested fields", () => {
  const descriptor = Descriptor.createFrom({
    id: "extension-driver",
    form: [],
    capabilities: [],
    indexEditing: { types: [] },
  });
  const before = JSON.parse(JSON.stringify(descriptor));
  const restored = Descriptor.createFrom(descriptor);
  expect(JSON.parse(JSON.stringify(restored))).toEqual(before);
  expect(restored.id).toBe("extension-driver");
  expect(restored.indexEditing?.types).toEqual([]);
});
