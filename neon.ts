import { defineConfig } from "@neon/config/v1";

export default defineConfig({
  auth: false,
  buckets: {
    // EIA zip/xlsx archives and other private blobs
    excel: { access: "private" },
  },
  branch: (branch) => {
    if (branch.isDefault) {
      return {};
    }
    if (!branch.exists) {
      return { ttl: "7d" };
    }
    return {};
  },
});
