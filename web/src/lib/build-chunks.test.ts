/**
 * Unit tests for build chunking logic.
 *
 * Tests the chunkNameForModuleId function to ensure proper
 * module-to-chunk mapping for both vendor and internal modules.
 *
 * @module lib/build-chunks.test
 */
import { describe, it, expect } from "vitest";
import { chunkNameForModuleId } from "./build-chunks";

describe("chunkNameForModuleId", () => {
  describe("vendor chunks", () => {
    it("should return vendor-react for react paths", () => {
      expect(chunkNameForModuleId("/node_modules/react/index.js")).toBe(
        "vendor-react",
      );
      expect(chunkNameForModuleId("/node_modules/react-dom/client.js")).toBe(
        "vendor-react",
      );
      expect(chunkNameForModuleId("C:\\node_modules\\react\\index.js")).toBe(
        "vendor-react",
      );
    });

    it("should return vendor-onboarding for joyride and deps", () => {
      expect(
        chunkNameForModuleId("/node_modules/react-joyride/dist/index.js"),
      ).toBe("vendor-onboarding");
      expect(chunkNameForModuleId("/node_modules/react-floater/index.js")).toBe(
        "vendor-onboarding",
      );
      expect(
        chunkNameForModuleId("/node_modules/react-focus-lock/index.js"),
      ).toBe("vendor-onboarding");
      expect(chunkNameForModuleId("/node_modules/focus-lock/index.js")).toBe(
        "vendor-onboarding",
      );
      expect(chunkNameForModuleId("/node_modules/tabbable/index.js")).toBe(
        "vendor-onboarding",
      );
    });

    it("should return vendor-ui for cmdk and radix-ui", () => {
      expect(chunkNameForModuleId("/node_modules/cmdk/dist/index.mjs")).toBe(
        "vendor-ui",
      );
      expect(
        chunkNameForModuleId("/node_modules/@radix-ui/react-dialog/index.js"),
      ).toBe("vendor-ui");
      expect(
        chunkNameForModuleId(
          "/node_modules/@radix-ui/react-focus-scope/index.js",
        ),
      ).toBe("vendor-ui");
    });

    it("should return undefined for other node_modules", () => {
      expect(chunkNameForModuleId("/node_modules/lodash/index.js")).toBe(
        undefined,
      );
      expect(chunkNameForModuleId("/node_modules/axios/index.js")).toBe(
        undefined,
      );
    });
  });

  describe("internal feature chunks", () => {
    it("should return feature-api for api directory", () => {
      expect(chunkNameForModuleId("/src/api/types.gen.ts")).toBe("feature-api");
      expect(chunkNameForModuleId("/src/api/sdk.gen.ts")).toBe("feature-api");
      expect(chunkNameForModuleId("/src/api/index.ts")).toBe("feature-api");
    });

    it("should return feature-shared for cross-feature lib modules", () => {
      expect(chunkNameForModuleId("/src/lib/formatting.ts")).toBe(
        "feature-shared",
      );
      expect(chunkNameForModuleId("/src/lib/input-parsing.ts")).toBe(
        "feature-shared",
      );
      expect(chunkNameForModuleId("/src/lib/batch-utils.ts")).toBe(
        "feature-shared",
      );
      expect(chunkNameForModuleId("/src/lib/watch-utils.ts")).toBe(
        "feature-shared",
      );
      expect(chunkNameForModuleId("/src/lib/webhook-utils.ts")).toBe(
        "feature-shared",
      );
    });

    it("should return feature-results for results components", () => {
      expect(
        chunkNameForModuleId("/src/components/results/ResultsExplorer.tsx"),
      ).toBe("feature-results");
      expect(
        chunkNameForModuleId("/src/components/results/ResultsViewer.tsx"),
      ).toBe("feature-results");
    });

    it("should return feature-templates for template components", () => {
      expect(
        chunkNameForModuleId("/src/components/templates/TemplateForm.tsx"),
      ).toBe("feature-templates");
      expect(
        chunkNameForModuleId("/src/components/TemplatePerformance.tsx"),
      ).toBe("feature-templates");
      expect(
        chunkNameForModuleId("/src/components/TemplateABTestManager.tsx"),
      ).toBe("feature-templates");
      expect(
        chunkNameForModuleId("/src/components/VisualSelectorBuilder.tsx"),
      ).toBe("feature-templates");
    });

    it("should return feature-feeds for feeds components", () => {
      expect(
        chunkNameForModuleId("/src/components/feeds/FeedManager.tsx"),
      ).toBe("feature-feeds");
      expect(chunkNameForModuleId("/src/components/feeds/FeedList.tsx")).toBe(
        "feature-feeds",
      );
    });

    it("should return feature-watches for watches components", () => {
      expect(
        chunkNameForModuleId("/src/components/watches/WatchManager.tsx"),
      ).toBe("feature-watches");
      expect(
        chunkNameForModuleId("/src/components/watches/WatchDetail.tsx"),
      ).toBe("feature-watches");
    });

    it("should return feature-export-schedules for export schedule components", () => {
      expect(
        chunkNameForModuleId(
          "/src/components/export-schedules/ExportScheduleManager.tsx",
        ),
      ).toBe("feature-export-schedules");
    });

    it("should return feature-webhooks for webhooks components", () => {
      expect(
        chunkNameForModuleId("/src/components/webhooks/WebhookConfig.tsx"),
      ).toBe("feature-webhooks");
    });

    it("should return feature-chains for chains components", () => {
      expect(
        chunkNameForModuleId("/src/components/chains/ChainBuilder.tsx"),
      ).toBe("feature-chains");
    });

    it("should return feature-batches for batches components", () => {
      expect(
        chunkNameForModuleId("/src/components/batches/BatchForm.tsx"),
      ).toBe("feature-batches");
    });

    it("should return feature-presets for presets components", () => {
      expect(
        chunkNameForModuleId("/src/components/presets/PresetManager.tsx"),
      ).toBe("feature-presets");
    });

    it("should return feature-jobs for jobs components", () => {
      expect(
        chunkNameForModuleId("/src/components/jobs/JobSubmissionContainer.tsx"),
      ).toBe("feature-jobs");
      expect(chunkNameForModuleId("/src/components/jobs/ScrapeForm.tsx")).toBe(
        "feature-jobs",
      );
    });
  });

  describe("cross-platform path handling", () => {
    it("should normalize Windows-style paths", () => {
      // In test files, single backslash in source becomes literal backslash at runtime
      expect(
        chunkNameForModuleId("C:\\project\\node_modules\\react\\index.js"),
      ).toBe("vendor-react");
      expect(
        chunkNameForModuleId(
          "C:\\project\\src\\components\\results\\Explorer.tsx",
        ),
      ).toBe("feature-results");
    });

    it("should handle Unix-style paths", () => {
      expect(
        chunkNameForModuleId("/home/project/node_modules/react/index.js"),
      ).toBe("vendor-react");
      expect(
        chunkNameForModuleId(
          "/home/project/src/components/results/Explorer.tsx",
        ),
      ).toBe("feature-results");
    });
  });

  describe("edge cases", () => {
    it("should return undefined for non-source files", () => {
      expect(chunkNameForModuleId("/some/random/path.js")).toBe(undefined);
      expect(chunkNameForModuleId("/src/styles/main.css")).toBe(undefined);
    });

    it("should handle empty strings", () => {
      expect(chunkNameForModuleId("")).toBe(undefined);
    });
  });
});
