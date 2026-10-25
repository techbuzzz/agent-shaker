/**
 * The plugin's own version.
 *
 * It is announced to the server so an operator can tell which build of the
 * client an agent is running. A constant rather than a read of package.json
 * because the package is published as compiled JavaScript with no JSON beside
 * it, and a test asserts the two agree.
 */
export const PLUGIN_VERSION = "0.1.0";

/** The npm package name, which is also the host's key for this plugin. */
export const PLUGIN_PACKAGE = "@techbuzzz/opencode-agent-shaker";
