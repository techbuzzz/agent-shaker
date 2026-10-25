import { Plugin } from "@opencode/plugin";
import type { Hooks, PluginInput, PluginModule, PluginOptions } from "@opencode-ai/plugin";
import { createV1Hooks } from "./adapters/v1.js";
import { setupV2 } from "./adapters/v2.js";
import type { AgentShakerOptions } from "./core/config.js";

/**
 * One package, two host APIs.
 *
 * The v1 and v2 plugin APIs are not renames of each other and the host does
 * not translate between them, so each adapter is written against its own
 * contract and both drive the same coordination core. The dual export lets a
 * single installed package serve an OpenCode host on either API.
 */
const plugin = {
  ...Plugin.define({
    id: "agent-shaker",
    setup: async (ctx) => setupV2(ctx),
  }),
  server: async (input: PluginInput, options?: PluginOptions): Promise<Hooks> => createV1Hooks(input, options),
} satisfies PluginModule & Plugin.Plugin;

export default plugin;

export type { AgentShakerOptions };
