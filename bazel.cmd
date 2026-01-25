:<<"::CMDLITERAL"
@ECHO OFF
GOTO :CMDSCRIPT
::CMDLITERAL

# Hermetic Bazelisk wrapper for Unix (Linux/macOS)
# Downloads and caches Bazelisk, enabling builds without system-wide installation.

set -eu

root="$(cd "$(dirname "$0")"; pwd)"
bazelisk_version=$(cat "$root/.bazeliskversion")
os=$(uname)
arch=$(uname -m)

case $os in
    Linux)
        os="linux"
        target_dir="${XDG_CACHE_HOME:-$HOME/.cache}/bazelle/bazelisk"
        ;;
    Darwin)
        os="darwin"
        target_dir="$HOME/Library/Caches/bazelle/bazelisk"
        ;;
    *)
        echo "Unsupported OS: $os" >&2
        exit 1
        ;;
esac

case $arch in
    x86_64)
        arch="amd64"
        ;;
    arm64|aarch64)
        arch="arm64"
        ;;
    *)
        echo "Unsupported architecture: $arch" >&2
        exit 1
        ;;
esac

binary_path="$target_dir/bazelisk-$bazelisk_version-${os}-${arch}"
mkdir -p "$target_dir"

if [ ! -x "$binary_path" ]; then
    download_url="https://github.com/bazelbuild/bazelisk/releases/download/v$bazelisk_version/bazelisk-${os}-${arch}"
    echo "Downloading Bazelisk v$bazelisk_version..." >&2
    echo "  $download_url" >&2
    curl -fsSL -o "$binary_path.tmp.$$" "$download_url"
    mv "$binary_path.tmp.$$" "$binary_path"
    chmod +x "$binary_path"
fi

exec "$binary_path" "$@"

:CMDSCRIPT
REM Hermetic Bazelisk wrapper for Windows
REM Downloads and caches Bazelisk, enabling builds without system-wide installation.

setlocal

set /p BAZELISK_VERSION=<"%~dp0.bazeliskversion"
set BAZELISK_TARGET_DIR=%LOCALAPPDATA%\bazelle\bazelisk
set BAZELISK_TARGET_FILE=%BAZELISK_TARGET_DIR%\bazelisk-%BAZELISK_VERSION%-windows-%PROCESSOR_ARCHITECTURE%.exe
set POWERSHELL=%SystemRoot%\system32\WindowsPowerShell\v1.0\powershell.exe
set POWERSHELL_COMMAND= ^
$ErrorActionPreference = \"Stop\"; ^
$ProgressPreference = \"SilentlyContinue\"; ^
Set-StrictMode -Version 3.0; ^
 ^
$arch = \"%PROCESSOR_ARCHITECTURE%\".ToLower(); ^
if ($arch -eq \"amd64\") { $arch = \"amd64\" } ^
elseif ($arch -eq \"arm64\") { $arch = \"arm64\" } ^
else { Write-Error \"Unsupported architecture: $arch\"; exit 1 }; ^
 ^
$BazeliskUrl = \"https://github.com/bazelbuild/bazelisk/releases/download/v%BAZELISK_VERSION%/bazelisk-windows-$arch.exe\"; ^
New-Item -ItemType Directory -Path \"%BAZELISK_TARGET_DIR%\" -Force | Out-Null; ^
 ^
$randomSuffix = [System.IO.Path]::GetRandomFileName(); ^
$tmpFile = \"%BAZELISK_TARGET_FILE%-$randomSuffix\"; ^
 ^
Write-Host \"Downloading Bazelisk v%BAZELISK_VERSION%...\"; ^
Write-Host \"  $BazeliskUrl\"; ^
$Web_client = New-Object System.Net.WebClient; ^
$Web_client.DownloadFile($BazeliskUrl, $tmpFile); ^
 ^
Move-Item -Path $tmpFile -Destination \"%BAZELISK_TARGET_FILE%\" -Force;

IF NOT EXIST "%BAZELISK_TARGET_FILE%" "%POWERSHELL%" -nologo -noprofile -Command %POWERSHELL_COMMAND% >&2

"%BAZELISK_TARGET_FILE%" %*
exit /B %ERRORLEVEL%
