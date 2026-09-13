$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go get go.opentelemetry.io/otel go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/sdk/trace go.opentelemetry.io/otel/sdk/resource go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp go.opentelemetry.io/contrib/bridges/otelslog go.opentelemetry.io/otel/propagation go.opentelemetry.io/otel/semconv/v1.26.0 *>&1 | Out-String
Write-Host $out
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
