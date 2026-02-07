@echo off
setlocal enabledelayedexpansion

if "%PROCESSOR_ARCHITECTURE%" == "x86" (
  echo WARNING: Your environment is 32-bit. Not all features are supported. >&2
  set arch=32
) else if "%PROCESSOR_ARCHITECTURE%" == "AMD64" (
  set arch=64u
  set arch2=x64
) else if "%PROCESSOR_ARCHITECTURE%" == "ARM64" (
  set arch=64a
  set arch2=arm64
) else (
  exit /b 1 
)

@REM BusyBox for Windows https://frippery.org/busybox/index.html
@REM Release Notes https://frippery.org/busybox/release-notes/index.html
@REM Index of /files/busybox https://frippery.org/files/busybox/?C=M;O=D
set ver=FRP-5857-g3681e397f
set cmd_name=busybox.exe
set cache_dir_path=%USERPROFILE%\.cache\task-sh\busybox@%ver%
if not exist !cache_dir_path! (
  mkdir "!cache_dir_path!"
)
set cmd_path=!cache_dir_path!\!cmd_name!
if not exist !cmd_path! (
  echo Downloading BusyBox for Windows. >&2
  curl.exe --fail --location --output "!cmd_path!" https://frippery.org/files/busybox/busybox-w!arch!-!ver!.exe || exit /b !ERRORLEVEL!
)
if not exist !cache_dir_path!\sh.exe (
  !cmd_path! --install !cache_dir_path!
)
@REM Shell-command ready (a5f342b)

@REM Prepend path
set PATH=!cache_dir_path!;%PATH%s

@REM Releases · jdx/mise https://github.com/jdx/mise/releases
set ver=2026.1.12
set cache_dir_path=%USERPROFILE%\.cache\task-sh\mise@%ver%
if not exist !cache_dir_path! (
  mkdir "!cache_dir_path!"
)
set cmd_name=mise.exe
set cmd_path=!cache_dir_path!\!cmd_name!
if not exist !cmd_path! (
  echo Downloading Mise for Windows. >&2
  set zip_path=!cache_dir_path!\mise.zip
  curl.exe --fail --location --output "!zip_path!" https://github.com/jdx/mise/releases/download/v!ver!/mise-v!ver!-windows-!arch2!.zip || exit /b !ERRORLEVEL!
  unzip.exe "!zip_path!" -d !cache_dir_path!\work || exit /b !ERRORLEVEL!
  mv.exe -f !cache_dir_path!\work\mise\bin\mise.exe "!cache_dir_path!"
  rm.exe -f "!zip_path!"
  rm.exe -fr !cache_dir_path!\work
)
!cmd_path! %* || exit /b !ERRORLEVEL!

endlocal
