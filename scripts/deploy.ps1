#!/usr/bin/env pwsh

[CmdletBinding()]
param(
    [string]$SshHost,
    [string]$SshUser,
    [ValidateRange(1, 65535)]
    [int]$SshPort = 22,
    [string]$RemoteStack = "/opt/dockpanel/stacks/easyforumgo",
    [string]$Branch = "dev",
    [string]$EnvFile,
    [switch]$SkipPublicCheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Read-RequiredValue {
    param(
        [string]$Prompt,
        [string]$CurrentValue
    )

    if (-not [string]::IsNullOrWhiteSpace($CurrentValue)) {
        return $CurrentValue.Trim()
    }

    do {
        $value = Read-Host $Prompt
    } while ([string]::IsNullOrWhiteSpace($value))

    return $value.Trim()
}

function Assert-Command {
    param([string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found."
    }
}

function Invoke-External {
    param(
        [string]$Description,
        [scriptblock]$Command
    )

    Write-Host "==> $Description"
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Description failed with exit code $LASTEXITCODE."
    }
}

function Send-TextFile {
    param(
        [string]$Target,
        [int]$Port,
        [string]$RemotePath,
        [string]$Content
    )

    $encoded = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($Content))
    $encoded | & ssh -p $Port $Target "umask 077; base64 -d > '$RemotePath'"
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to upload $RemotePath."
    }
}

function Get-EnvValue {
    param(
        [string]$Content,
        [string]$Name
    )

    $match = [regex]::Match(
        $Content,
        "(?m)^\s*$([regex]::Escape($Name))\s*=\s*(.*)\s*$"
    )
    if (-not $match.Success) {
        return ""
    }
    return $match.Groups[1].Value.Trim().Trim('"').Trim("'")
}

Assert-Command "pwsh"
Assert-Command "ssh"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($EnvFile)) {
    $EnvFile = Join-Path $repositoryRoot ".env.prod"
} elseif (-not [IO.Path]::IsPathRooted($EnvFile)) {
    $EnvFile = Join-Path $repositoryRoot $EnvFile
}
$EnvFile = [IO.Path]::GetFullPath($EnvFile)

if (-not (Test-Path -LiteralPath $EnvFile -PathType Leaf)) {
    throw "Production environment file not found: $EnvFile"
}

$SshHost = Read-RequiredValue "SSH host or IP" $SshHost
$SshUser = Read-RequiredValue "SSH user" $SshUser

if ($SshHost -notmatch '^[a-zA-Z0-9._:-]+$') {
    throw "SSH host contains unsupported characters."
}
if ($SshUser -notmatch '^[a-zA-Z0-9._-]+$') {
    throw "SSH user contains unsupported characters."
}
if ($RemoteStack -notmatch '^/[a-zA-Z0-9._/-]+$') {
    throw "Remote stack path must be an absolute path without spaces."
}
if ($Branch -notmatch '^[a-zA-Z0-9._/-]+$') {
    throw "Branch contains unsupported characters."
}

$envContent = [IO.File]::ReadAllText($EnvFile)
$appEnv = Get-EnvValue $envContent "APP_ENV"
$basePath = Get-EnvValue $envContent "APP_BASE_PATH"
$publicURL = Get-EnvValue $envContent "APP_PUBLIC_URL"

if ($appEnv -ne "prod") {
    throw "APP_ENV must be 'prod' in $EnvFile."
}
if ($publicURL -notmatch '^https://') {
    throw "APP_PUBLIC_URL must use HTTPS in $EnvFile."
}
if ([string]::IsNullOrWhiteSpace($basePath)) {
    $basePath = "/"
}

$target = "$SshUser@$SshHost"
$remoteEnv = "/tmp/easyforumgo-env-$PID"
$remoteScript = "/tmp/easyforumgo-deploy-$PID.sh"
$repositoryURL = "https://github.com/YajiTV/EasyForumGo.git"

$deployScript = @'
#!/bin/bash
set -euo pipefail

STACK="$1"
REPOSITORY="$2"
BRANCH="$3"
SOURCE_ENV="$4"
BASE_PATH="$5"

cleanup() {
  rm -f "$SOURCE_ENV" "$0"
}
trap cleanup EXIT

echo "==> Checking server prerequisites"
command -v git >/dev/null
command -v docker >/dev/null
docker compose version >/dev/null
docker network inspect web >/dev/null

echo "==> Preparing stack directory"
sudo mkdir -p "$STACK"
sudo chown "$(id -un):$(id -gn)" "$STACK"

if [ ! -d "$STACK/.git" ]; then
  git clone --branch "$BRANCH" "$REPOSITORY" "$STACK"
else
  cd "$STACK"
  if [ -n "$(git status --porcelain)" ]; then
    echo "Remote repository contains local changes: $STACK" >&2
    exit 1
  fi
  git fetch origin
  git checkout "$BRANCH"
  git pull --ff-only origin "$BRANCH"
fi

echo "==> Installing production environment"
sudo install -o root -g root -m 600 "$SOURCE_ENV" "$STACK/.env.prod"

echo "==> Validating and deploying application"
cd "$STACK"
COMPOSE_ARGS=(--env-file .env.prod -f docker/docker-compose.yml -f docker/docker-compose.prod.yml)
sudo docker compose "${COMPOSE_ARGS[@]}" config >/tmp/easyforumgo-compose-rendered.yml
sudo docker compose "${COMPOSE_ARGS[@]}" up -d --build
sudo docker compose "${COMPOSE_ARGS[@]}" ps

echo "==> Checking application from inside its container"
sudo docker compose "${COMPOSE_ARGS[@]}" exec -T forum \
  wget -qO- "http://127.0.0.1:8080${BASE_PATH%/}/" >/dev/null

echo "==> Deployment completed"
'@

Write-Host ""
Write-Host "Deployment target : $target`:$SshPort"
Write-Host "Remote stack      : $RemoteStack"
Write-Host "Git branch        : $Branch"
Write-Host "Public URL        : $publicURL"
Write-Host "Environment file  : $EnvFile"
Write-Host ""

$confirmation = Read-Host "Deploy EasyForumGo? Type 'deploy' to continue"
if ($confirmation -ne "deploy") {
    Write-Host "Deployment cancelled."
    exit 0
}

try {
    Write-Host "==> Uploading deployment files"
    Send-TextFile $target $SshPort $remoteEnv $envContent
    Send-TextFile $target $SshPort $remoteScript $deployScript

    $remoteCommand = "bash '$remoteScript' '$RemoteStack' '$repositoryURL' '$Branch' '$remoteEnv' '$basePath'"
    Invoke-External "Deploying on remote server" {
        & ssh -t -p $SshPort $target $remoteCommand
    }
} catch {
    & ssh -p $SshPort $target "rm -f '$remoteEnv' '$remoteScript'" 2>$null
    throw
}

if (-not $SkipPublicCheck) {
    Write-Host "==> Checking public URL"
    try {
        $response = Invoke-WebRequest -Uri "$($publicURL.TrimEnd('/'))/" -MaximumRedirection 5
        Write-Host "Public check succeeded with HTTP $($response.StatusCode)."
    } catch {
        Write-Warning "The application was deployed, but the public check failed: $($_.Exception.Message)"
        Write-Warning "Verify the reverse-proxy route for $publicURL."
    }
}

Write-Host "EasyForumGo deployment finished."
