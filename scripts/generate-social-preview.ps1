#requires -Version 5.1
<#
.SYNOPSIS
  Compose a 1280x640 social preview PNG from:
    - images/logo-clean.png  (icon + wordmark + tagline)
    - images/icon-clean.png  (icon only)

  Output:
    docs-site/public/social-preview.png   (used as OG / Twitter image)
    images/social-preview.png            (project-wide, ready for upload as
                                          the GitHub repo "Social preview")
#>

param()

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$root = (Resolve-Path '.').Path
$iconPath = Join-Path $root 'images\icon-clean.png'
$logoPath = Join-Path $root 'images\logo-clean.png'
if (-not (Test-Path $iconPath)) { throw "missing $iconPath" }
if (-not (Test-Path $logoPath)) { throw "missing $logoPath" }

$W = 1280
$H = 640
$bmp = New-Object System.Drawing.Bitmap($W, $H, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$g = [System.Drawing.Graphics]::FromImage($bmp)
try {
  $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
  $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
  $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
  $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit
  $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality

  # Base: vertical gradient backdrop slate-950 -> slate-900.
  $backBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    (New-Object System.Drawing.Rectangle(0, 0, $W, $H)),
    [System.Drawing.Color]::FromArgb(255, 2, 6, 12),
    [System.Drawing.Color]::FromArgb(255, 15, 23, 36),
    [System.Drawing.Drawing2D.LinearGradientMode]::Vertical
  )
  $g.FillRectangle($backBrush, 0, 0, $W, $H)
  $backBrush.Dispose()

  # Subtle, focused brand-color highlights (much smaller + lower alpha so the
  # icon and logo stay in the foreground).
  $teal = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(35, 45, 212, 191))
  $g.FillEllipse($teal, 800, -160, 540, 540)
  $teal.Dispose()
  $purple = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(35, 168, 85, 247))
  $g.FillEllipse($purple, $W - 380, $H - 240, 420, 420)
  $purple.Dispose()
  $cyan = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(28, 34, 211, 238))
  $g.FillEllipse($cyan, -180, $H - 280, 360, 360)
  $cyan.Dispose()

  # Left half: icon-only mark.
  $iconImg = [System.Drawing.Image]::FromFile($iconPath)
  try {
    $iconSize = 460
    $iconX = 110
    $iconY = [int](($H - $iconSize) / 2)
    $g.DrawImage($iconImg, $iconX, $iconY, $iconSize, $iconSize)
  } finally { $iconImg.Dispose() }

  # Right half: full logo (icon + wordmark + tagline).
  $logoImg = [System.Drawing.Image]::FromFile($logoPath)
  try {
    $logoH = 360
    $ratio = $logoImg.Width / $logoImg.Height
    $logoW = [int][Math]::Round($logoH * $ratio)
    $logoX = 620
    $logoY = [int](($H - $logoH) / 2) - 30
    $g.DrawImage($logoImg, $logoX, $logoY, $logoW, $logoH)
  } finally { $logoImg.Dispose() }

  # Below the logo: small label tying the preview to the GitHub repo URL.
  $urlFont = New-Object System.Drawing.Font('Segoe UI, Inter, system-ui, sans-serif', 22, [System.Drawing.FontStyle]::Regular)
  try {
    $urlBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 148, 163, 184))
    $g.DrawString('github.com/techbuzzz/agent-shaker', $urlFont, $urlBrush, 625, 540)
    $urlBrush.Dispose()
  } finally { $urlFont.Dispose() }
} finally {
  $g.Dispose()
}

$outA = Join-Path $root 'docs-site\public\social-preview.png'
$outB = Join-Path $root 'images\social-preview.png'
$bmp.Save($outA, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Copy-Item -Path $outA -Destination $outB -Force
Write-Host "wrote: $outA"
Write-Host "wrote: $outB"