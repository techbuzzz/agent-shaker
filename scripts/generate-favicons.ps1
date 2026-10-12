#requires -Version 5.1
<#
.SYNOPSIS
  Generate favicon variants from images/icon-clean.png into:
   - docs-site/public/    (deployed via GitHub Pages)
   - images/favicon/     (project-wide brand assets)

  Sizes produced:
   - favicon-16x16.png, favicon-32x32.png
   - apple-touch-icon.png (180x180)
   - android-chrome-192x192.png, android-chrome-512x512.png
   - favicon.ico (16, 32, 48) — multi-resolution

  PNG source is preserved as docs-site/public/icon.png (the high-res header art)
  and images/icon-clean.png / images/icon.png (canonical brand mark).
#>

param(
  [string]$Source = 'images\icon-clean.png'
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$root   = (Resolve-Path '.').Path
$srcPng = Join-Path $root $Source
if (-not (Test-Path $srcPng)) { throw "source png not found: $srcPng" }

$dstDocs   = Join-Path $root 'docs-site\public'
$dstBrand  = Join-Path $root 'images\favicon'

function Resize-Png {
  param([string]$InPath,[int]$Size,[string]$OutPath)
  $src = [System.Drawing.Image]::FromFile($InPath)
  try {
    $dst = New-Object System.Drawing.Bitmap($Size, $Size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    try {
      $g = [System.Drawing.Graphics]::FromImage($dst)
      try {
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
        $g.Clear([System.Drawing.Color]::Transparent)
        $g.DrawImage($src, 0, 0, $Size, $Size)
      } finally { $g.Dispose() }
      $dir = Split-Path -Parent $OutPath
      if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
      $dst.Save($OutPath, [System.Drawing.Imaging.ImageFormat]::Png)
    } finally { $dst.Dispose() }
  } finally { $src.Dispose() }
  Write-Host ("  {0} -> {1}" -f $Size, $OutPath)
}

function Write-Ico {
  param([string]$InPath,[int[]]$Sizes,[string]$OutPath)
  # Build a multi-image ICO: header + dir entries + PNG-compressed images.
  $images = foreach ($s in $Sizes) {
    $src = [System.Drawing.Image]::FromFile($InPath)
    try {
      $bmp = New-Object System.Drawing.Bitmap($s, $s, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
      try {
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        try {
          $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
          $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
          $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
          $g.Clear([System.Drawing.Color]::Transparent)
          $g.DrawImage($src, 0, 0, $s, $s)
        } finally { $g.Dispose() }
        $ms = New-Object System.IO.MemoryStream
        $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
        @{ Size = $s; Bytes = $ms.ToArray() }
      } finally { $bmp.Dispose() }
    } finally { $src.Dispose() }
  }

  $count = $images.Count
  $out = New-Object System.IO.MemoryStream
  $bw = New-Object System.IO.BinaryWriter($out)
  try {
    # ICONDIR: reserved=0, type=1, count
    $bw.Write([uint16]0)
    $bw.Write([uint16]1)
    $bw.Write([uint16]$count)
    # Each entry is 16 bytes. Image data follows at offset = 6 + count*16.
    $dataStart = $dataStartOffset = 6 + 16 * $count
    foreach ($img in $images) {
      $s = $img.Size
      $bytes = $img.Bytes
      $bw.Write([byte]($(if ($s -ge 256) { 0 } else { $s })))   # width (0 == 256)
      $bw.Write([byte]($(if ($s -ge 256) { 0 } else { $s })))   # height (0 == 256)
      $bw.Write([byte]0)                                        # palette
      $bw.Write([byte]0)                                        # reserved
      $bw.Write([uint16]1)                                      # planes
      $bw.Write([uint16]32)                                     # bit count
      $bw.Write([uint32]$bytes.Length)                          # size
      $bw.Write([uint32]$dataStartOffset)                       # offset
      $dataStartOffset += $bytes.Length
    }
    foreach ($img in $images) { $bw.Write($img.Bytes) }
    $bw.Flush()
    [System.IO.File]::WriteAllBytes($OutPath, $out.ToArray())
  } finally {
    $bw.Dispose()
    $out.Dispose()
  }
  Write-Host ("  ico[{0}] -> {1}" -f ($Sizes -join ','), $OutPath)
}

Write-Host "Generating favicon variants from $srcPng ..."

# docs-site/public/
Resize-Png -InPath $srcPng -Size 16  -OutPath (Join-Path $dstDocs 'favicon-16x16.png')
Resize-Png -InPath $srcPng -Size 32  -OutPath (Join-Path $dstDocs 'favicon-32x32.png')
Resize-Png -InPath $srcPng -Size 180 -OutPath (Join-Path $dstDocs 'apple-touch-icon.png')
Resize-Png -InPath $srcPng -Size 192 -OutPath (Join-Path $dstDocs 'android-chrome-192x192.png')
Resize-Png -InPath $srcPng -Size 512 -OutPath (Join-Path $dstDocs 'android-chrome-512x512.png')
Write-Ico   -InPath $srcPng -Sizes @(16,32,48) -OutPath (Join-Path $dstDocs 'favicon.ico')

# images/favicon/ (project-wide brand set)
Resize-Png -InPath $srcPng -Size 16  -OutPath (Join-Path $dstBrand 'favicon-16x16.png')
Resize-Png -InPath $srcPng -Size 32  -OutPath (Join-Path $dstBrand 'favicon-32x32.png')
Resize-Png -InPath $srcPng -Size 180 -OutPath (Join-Path $dstBrand 'apple-touch-icon.png')
Resize-Png -InPath $srcPng -Size 192 -OutPath (Join-Path $dstBrand 'android-chrome-192x192.png')
Resize-Png -InPath $srcPng -Size 512 -OutPath (Join-Path $dstBrand 'android-chrome-512x512.png')
Write-Ico   -InPath $srcPng -Sizes @(16,32,48) -OutPath (Join-Path $dstBrand 'favicon.ico')

Write-Host "Done."