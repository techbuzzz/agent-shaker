// One-shot transformation: move the MCP tool registry out of
// internal/mcp/handler.go into internal/mcp/tools.go.
//
// The advertised schemas and the dispatch that serves them were two lists kept
// in step by hand, in a file of three thousand lines. A tool could be
// advertised and never dispatched, or dispatched and never advertised, and
// nothing noticed until an agent called it.
//
// The script relocates the schema block verbatim and derives the dispatch map
// from the switch it replaces, so the transformation is mechanical and the
// compiler and the test suite are what judge the result.
//
// Run once, from the repository root:
//
//   node scripts/extract-tool-registry.mjs
//
// It refuses to run twice: the second run finds no anchors and stops.

import { readFileSync, writeFileSync } from 'node:fs'

const handlerPath = 'internal/mcp/handler.go'
const toolsPath = 'internal/mcp/tools.go'

let src = readFileSync(handlerPath, 'utf8')

// --- the types -------------------------------------------------------------

const typesBlock = `type Tool struct {
	Name        string      \`json:"name"\`
	Description string      \`json:"description,omitempty"\`
	InputSchema InputSchema \`json:"inputSchema"\`
}

type InputSchema struct {
	Type       string                 \`json:"type"\`
	Properties map[string]interface{} \`json:"properties,omitempty"\`
	Required   []string               \`json:"required,omitempty"\`
}
`

if (!src.includes(typesBlock)) {
  throw new Error('the Tool/InputSchema types are not where they were; run this once')
}
src = src.replace(typesBlock, '')

// --- the tool list ---------------------------------------------------------

const listPrefix = '\ttools := []Tool{'
const listStart = src.indexOf(listPrefix)
if (listStart < 0) throw new Error('the tool list is not where it was; run this once')

const listEndAnchor = src.indexOf('\n\n\trequireRequiredStrings(tools)')
if (listEndAnchor < 0) throw new Error('the tool list end anchor is missing')

const listInner = src.slice(listStart + listPrefix.length, listEndAnchor)
const listBody = listInner.slice(0, listInner.lastIndexOf('\n\t}'))

src =
  src.slice(0, listStart) +
  '\ttools := toolDefinitions()' +
  src.slice(listEndAnchor)

// --- the dispatch switch ---------------------------------------------------

const switchStart = src.indexOf('\tswitch callParams.Name {')
if (switchStart < 0) throw new Error('the dispatch switch is not where it was; run this once')

const switchEndAnchor = src.indexOf('\n\n\t// A tool that returns IsError')
if (switchEndAnchor < 0) throw new Error('the dispatch end anchor is missing')

const switchBody = src.slice(switchStart, switchEndAnchor)

const dispatchCall = `\tresultText, isError, rpcErr := h.dispatchTool(callParams.Name, callParams.Arguments, ctx)
\tif rpcErr != nil {
\t\tspan.RecordError(fmt.Errorf("unknown tool: %s", callParams.Name))
\t\tspan.SetStatus(codes.Error, "unknown tool")
\t\treturn nil, rpcErr
\t}`

src =
  src.slice(0, switchStart) +
  dispatchCall +
  src.slice(switchEndAnchor)

// --- the executor map ------------------------------------------------------

const caseRe = /case "([a-z0-9_]+)":\n\t\tresultText, isError = (h\.execute[A-Za-z0-9_]+)\(([^)]*)\)\n/g

const entries = []
let match
while ((match = caseRe.exec(switchBody)) !== null) {
  const [, name, method, rawArgs] = match
  const args = rawArgs.trim()

  let params
  let call
  if (args === 'callParams.Arguments, ctx') {
    params = 'args map[string]interface{}, ctx MCPContext'
    call = `${method}(args, ctx)`
  } else if (args === 'callParams.Arguments') {
    params = 'args map[string]interface{}, _ MCPContext'
    call = `${method}(args)`
  } else if (args === 'ctx') {
    params = '_ map[string]interface{}, ctx MCPContext'
    call = `${method}(ctx)`
  } else if (args === '') {
    params = '_ map[string]interface{}, _ MCPContext'
    call = `${method}()`
  } else {
    throw new Error(`unrecognised call shape for ${name}: ${args}`)
  }

  entries.push(
    `\t"${name}": func(h *MCPHandler, ${params}) (string, bool) {\n` +
      `\t\treturn ${call}\n` +
      `\t},`,
  )
}

if (entries.length === 0) throw new Error('no dispatch cases were found')

// --- the new file ----------------------------------------------------------

const toolsFile = `package mcp

import (
\t"fmt"

\t"github.com/techbuzzz/agent-shaker/internal/models"
)

// The tool registry.
//
// The advertised schemas and the dispatch that serves them live here, in one
// file, and a test holds them in step: every name in the list has an executor,
// and every executor has a schema. They were two lists in a file of three
// thousand lines, and a tool could be advertised without being callable — or
// callable without being advertised — with nothing noticing until an agent
// called it and got "unknown tool".
${typesBlock}
// toolExecutor runs one MCP tool.
//
// The arguments and the connection context are always passed, and a tool that
// needs neither ignores them. One signature for every tool is what lets the
// registry be a list rather than three shapes.
type toolExecutor func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool)

// toolDefinitions returns every tool this server advertises.
func toolDefinitions() []Tool {
\treturn []Tool{${listBody}
\t}
}

// toolExecutors maps a tool name to its implementation.
//
// A map rather than a switch, so the set of callable tools is enumerable. A
// switch's case labels are not enumerable at runtime, and the test that keeps
// the advertised tools and the callable tools in step would have to parse Go
// source to do it.
var toolExecutors = map[string]toolExecutor{
${entries.join('\n')}
}

// dispatchTool runs one tool by name.
//
// An unknown name is a JSON-RPC error rather than a tool error: the client
// asked for something this server does not have.
func (h *MCPHandler) dispatchTool(name string, args map[string]interface{}, ctx MCPContext) (string, bool, *JSONRPCError) {
\texecutor, ok := toolExecutors[name]
\tif !ok {
\t\treturn "", false, &JSONRPCError{
\t\t\tCode:    -32601,
\t\t\tMessage: "Unknown tool",
\t\t\tData:    fmt.Sprintf("Tool not found: %s", name),
\t\t}
\t}

\tresultText, isError := executor(h, args, ctx)
\treturn resultText, isError, nil
}
`

writeFileSync(toolsPath, toolsFile)
writeFileSync(handlerPath, src)

console.log(`moved ${entries.length} tools into ${toolsPath}`)
