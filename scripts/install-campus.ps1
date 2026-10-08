# ============================================================
# 校园Memo 一键安装脚本（Windows PowerShell）
#
# 用法：右键「使用 PowerShell 运行」，或在 PowerShell 中执行：
#   irm https://raw.githubusercontent.com/<你的用户名>/campus-memos/main/scripts/install-campus.ps1 | iex
#
# 策略与 Linux 版一致：有 Docker 用 Docker，否则下载单二进制。
# ============================================================
$ErrorActionPreference = "Stop"

$AppName = "campus-memos"
$DataDir = if ($env:CAMPUS_DATA) { $env:CAMPUS_DATA } else { "$env:USERPROFILE\.campus-memos" }
$Port    = if ($env:CAMPUS_PORT) { $env:CAMPUS_PORT } else { 5230 }
# TODO(fork 后修改为你的仓库地址)
$Repo    = if ($env:CAMPUS_REPO) { $env:CAMPUS_REPO } else { "yourname/campus-memos" }

Write-Host "==> 校园Memo 一键安装"
Write-Host "    数据目录: $DataDir"
Write-Host "    访问端口: $Port"
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host "==> 检测到 Docker，使用容器方式安装"
    docker pull "ghcr.io/$Repo`:latest"
    docker rm -f $AppName 2>$null
    docker run -d --name $AppName -p "${Port}:5230" -v "${DataDir}:/var/opt/memos" --restart unless-stopped "ghcr.io/$Repo`:latest"
    Write-Host "==> 完成！浏览器打开: http://localhost:$Port"
    exit 0
}

Write-Host "==> 未检测到 Docker，尝试下载单二进制（无需安装数据库）"
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$Url = "https://github.com/$Repo/releases/latest/download/campus-memos_windows_$Arch.zip"
$Zip = "$env:TEMP\campus-memos.zip"
try {
    Invoke-WebRequest -Uri $Url -OutFile $Zip
    Expand-Archive -Force $Zip "$DataDir\bin"
    Start-Process -WindowStyle Hidden -FilePath "$DataDir\bin\campus-memos.exe" -ArgumentList "--data `"$DataDir`" --port $Port"
    Write-Host "==> 完成！浏览器打开: http://localhost:$Port"
    Write-Host "    数据在 $DataDir"
} catch {
    Write-Host "下载失败：$_"
    Write-Host "请先安装 Docker Desktop 后重跑本脚本，或到 https://github.com/$Repo/releases 手动下载。"
    exit 1
}
