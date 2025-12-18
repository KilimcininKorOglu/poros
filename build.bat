@echo off
setlocal enabledelayedexpansion

:: Poros - Modern Network Path Tracer
:: Build script for Windows

:: Colors (Windows 10+)
set "GREEN=[92m"
set "YELLOW=[93m"
set "RED=[91m"
set "CYAN=[96m"
set "NC=[0m"

:: Variables
set "BINARY_NAME=poros"
set "BINARY_DIR=bin"

:: Get version from git
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
if "%VERSION%"=="" set "VERSION=dev"

:: Get commit hash
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%i"
if "%COMMIT%"=="" set "COMMIT=unknown"

:: Get build date
for /f "tokens=*" %%i in ('powershell -command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'"') do set "DATE=%%i"

:: LDFLAGS
set "LDFLAGS=-ldflags "-s -w -X main.version=%VERSION% -X main.commit=%COMMIT% -X main.date=%DATE%""

:: Parse command
if "%1"=="" goto :build
if "%1"=="help" goto :help
if "%1"=="-h" goto :help
if "%1"=="--help" goto :help
if "%1"=="build" goto :build
if "%1"=="build-all" goto :build-all
if "%1"=="build-linux" goto :build-linux
if "%1"=="build-linux-arm64" goto :build-linux-arm64
if "%1"=="build-darwin" goto :build-darwin
if "%1"=="build-darwin-arm64" goto :build-darwin-arm64
if "%1"=="build-windows" goto :build-windows
if "%1"=="test" goto :test
if "%1"=="test-short" goto :test-short
if "%1"=="test-cover" goto :test-cover
if "%1"=="bench" goto :bench
if "%1"=="lint" goto :lint
if "%1"=="fmt" goto :fmt
if "%1"=="vet" goto :vet
if "%1"=="check" goto :check
if "%1"=="clean" goto :clean
if "%1"=="deps" goto :deps
if "%1"=="run" goto :run
if "%1"=="dev" goto :dev
if "%1"=="install" goto :install
if "%1"=="version" goto :version

echo %RED%Unknown command: %1%NC%
echo Run 'build.bat help' for usage
exit /b 1

:: ==================== BUILD TARGETS ====================

:build
echo %GREEN%Building %BINARY_NAME% for Windows...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" .\cmd\poros
if errorlevel 1 (
    echo %RED%Build failed%NC%
    exit /b 1
)
echo %GREEN%Built: %BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe%NC%
goto :eof

:dev
echo %CYAN%Building development version (no optimizations)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
go build -o "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" .\cmd\poros
if errorlevel 1 (
    echo %RED%Build failed%NC%
    exit /b 1
)
echo %GREEN%Built: %BINARY_DIR%\%BINARY_NAME%.exe%NC%
goto :eof

:install
echo %CYAN%Installing %BINARY_NAME% to GOPATH\bin...%NC%
go install %LDFLAGS% .\cmd\poros
if errorlevel 1 (
    echo %RED%Install failed%NC%
    exit /b 1
)
echo %GREEN%Installed to GOPATH\bin%NC%
goto :eof

:: ==================== CROSS-COMPILATION ====================

:build-all
echo %GREEN%Building for all platforms...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
call :build-windows
call :build-linux
call :build-linux-arm64
call :build-darwin
call :build-darwin-arm64
echo %GREEN%All builds complete!%NC%
goto :eof

:build-windows
echo %CYAN%Building for Windows (amd64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=windows
set GOARCH=amd64
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" .\cmd\poros
set GOOS=
set GOARCH=
if errorlevel 1 (
    echo %RED%Windows build failed%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe%NC%
goto :eof

:build-linux
echo %CYAN%Building for Linux (amd64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-linux-amd64" .\cmd\poros
set GOOS=
set GOARCH=
set CGO_ENABLED=
if errorlevel 1 (
    echo %RED%Linux build failed%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%BINARY_NAME%-linux-amd64%NC%
goto :eof

:build-linux-arm64
echo %CYAN%Building for Linux (arm64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=linux
set GOARCH=arm64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-linux-arm64" .\cmd\poros
set GOOS=
set GOARCH=
set CGO_ENABLED=
if errorlevel 1 (
    echo %RED%Linux ARM64 build failed%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%BINARY_NAME%-linux-arm64%NC%
goto :eof

:build-darwin
echo %CYAN%Building for macOS (amd64)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-darwin-amd64" .\cmd\poros
set GOOS=
set GOARCH=
set CGO_ENABLED=
if errorlevel 1 (
    echo %RED%macOS build failed%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%BINARY_NAME%-darwin-amd64%NC%
goto :eof

:build-darwin-arm64
echo %CYAN%Building for macOS (arm64/Apple Silicon)...%NC%
if not exist "%BINARY_DIR%" mkdir "%BINARY_DIR%"
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build %LDFLAGS% -o "%BINARY_DIR%\%BINARY_NAME%-darwin-arm64" .\cmd\poros
set GOOS=
set GOARCH=
set CGO_ENABLED=
if errorlevel 1 (
    echo %RED%macOS ARM64 build failed%NC%
    exit /b 1
)
echo   %GREEN%Created: %BINARY_DIR%\%BINARY_NAME%-darwin-arm64%NC%
goto :eof

:: ==================== TEST TARGETS ====================

:test
echo %CYAN%Running all tests...%NC%
go test -v -race -cover ./...
if errorlevel 1 (
    echo %RED%Tests failed%NC%
    exit /b 1
)
echo %GREEN%All tests passed%NC%
goto :eof

:test-short
echo %CYAN%Running short tests...%NC%
go test -v -short ./...
if errorlevel 1 (
    echo %RED%Tests failed%NC%
    exit /b 1
)
echo %GREEN%Short tests passed%NC%
goto :eof

:test-cover
echo %CYAN%Running tests with coverage...%NC%
go test -v -race -coverprofile=coverage.out ./...
if errorlevel 1 (
    echo %RED%Tests failed%NC%
    exit /b 1
)
go tool cover -html=coverage.out -o coverage.html
echo %GREEN%Coverage report: coverage.html%NC%
goto :eof

:bench
echo %CYAN%Running benchmarks...%NC%
go test -bench=. -benchmem ./...
echo %GREEN%Benchmarks complete%NC%
goto :eof

:: ==================== CODE QUALITY ====================

:lint
echo %CYAN%Running golangci-lint...%NC%
where golangci-lint >nul 2>&1
if errorlevel 1 (
    echo %YELLOW%golangci-lint not installed.%NC%
    echo Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    exit /b 1
)
golangci-lint run ./...
if errorlevel 1 (
    echo %RED%Linting failed%NC%
    exit /b 1
)
echo %GREEN%Linting passed%NC%
goto :eof

:fmt
echo %CYAN%Formatting code...%NC%
gofmt -s -w .
go mod tidy
echo %GREEN%Code formatted%NC%
goto :eof

:vet
echo %CYAN%Running go vet...%NC%
go vet ./...
if errorlevel 1 (
    echo %RED%go vet found issues%NC%
    exit /b 1
)
echo %GREEN%go vet passed%NC%
goto :eof

:check
echo %CYAN%Running all checks...%NC%
call :fmt
call :vet
call :lint
call :test-short
echo %GREEN%All checks passed%NC%
goto :eof

:: ==================== OTHER ====================

:deps
echo %CYAN%Downloading dependencies...%NC%
go mod download
go mod tidy
echo %GREEN%Dependencies ready%NC%
goto :eof

:clean
echo %CYAN%Cleaning build artifacts...%NC%
if exist "%BINARY_DIR%" rmdir /s /q "%BINARY_DIR%"
if exist "coverage.out" del "coverage.out"
if exist "coverage.html" del "coverage.html"
echo %GREEN%Cleaned%NC%
goto :eof

:run
call :build
if errorlevel 1 exit /b 1
echo %GREEN%Running %BINARY_NAME%...%NC%
echo.
"%BINARY_DIR%\%BINARY_NAME%-windows-amd64.exe" %2 %3 %4 %5 %6 %7 %8 %9
goto :eof

:version
echo %CYAN%Version Info:%NC%
echo   Version: %VERSION%
echo   Commit:  %COMMIT%
echo   Date:    %DATE%
goto :eof

:: ==================== HELP ====================

:help
echo.
echo %GREEN%Poros - Modern Network Path Tracer%NC%
echo %GREEN%===================================%NC%
echo.
echo Usage: build.bat [command] [args]
echo.
echo %YELLOW%Build:%NC%
echo   build              Build for Windows (default)
echo   build-all          Build for all platforms
echo   build-windows      Build for Windows (amd64)
echo   build-linux        Build for Linux (amd64)
echo   build-linux-arm64  Build for Linux (arm64)
echo   build-darwin       Build for macOS (amd64)
echo   build-darwin-arm64 Build for macOS (arm64/Apple Silicon)
echo   dev                Quick dev build (no optimizations)
echo   install            Install to GOPATH\bin
echo.
echo %YELLOW%Test:%NC%
echo   test               Run all tests with race detector
echo   test-short         Run short tests only
echo   test-cover         Run tests with coverage report
echo   bench              Run benchmarks
echo.
echo %YELLOW%Code Quality:%NC%
echo   lint               Run golangci-lint
echo   fmt                Format code
echo   vet                Run go vet
echo   check              Run fmt, vet, lint, test-short
echo.
echo %YELLOW%Other:%NC%
echo   deps               Download dependencies
echo   clean              Remove build artifacts
echo   run [args]         Build and run with arguments
echo   version            Show version info
echo   help               Show this help
echo.
echo %CYAN%Examples:%NC%
echo   build.bat                      Build for Windows
echo   build.bat build-all            Build all platforms
echo   build.bat run google.com       Build and trace google.com
echo   build.bat run -v 8.8.8.8       Build and trace with verbose
echo   build.bat test                 Run all tests
echo   build.bat check                Run all quality checks
echo.
echo %CYAN%Output:%NC%
echo   bin\poros-windows-amd64.exe    Windows binary
echo   bin\poros-linux-amd64          Linux binary
echo   bin\poros-linux-arm64          Linux ARM64 binary
echo   bin\poros-darwin-amd64         macOS Intel binary
echo   bin\poros-darwin-arm64         macOS Apple Silicon binary
echo.
goto :eof
