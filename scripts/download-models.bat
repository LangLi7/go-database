@echo off
REM go-database Model Downloader
REM Lädt die 3 empfohlenen Modelle für lokale Nutzung.
REM Erfordert: huggingface-cli (pip install huggingface-hub)
REM
REM Nutzung:
REM   scripts\download-models.bat          → Alle 3 Modelle
REM   scripts\download-models.bat ornith9  → Nur Ornith 1.0 9B
REM   scripts\download-models.bat ds14     → Nur DeepSeek 14B
REM   scripts\download-models.bat ornith35 → Nur Ornith 1.0 35B

setlocal enabledelayedexpansion

set MODELS_DIR=%USERPROFILE%\.lmstudio\models

if not "%1"=="" goto :%1
goto :all

:all
echo [go-database] Lade alle 3 Modelle...

:ornith9
echo.
echo === 1/3: Ornith 1.0 9B (Q4_K_M) - empfohlen für go-database ===
mkdir "%MODELS_DIR%\deepreinforce-ai\Ornith-1.0-9B-GGUF" 2>nul
huggingface-cli download deepreinforce-ai/Ornith-1.0-9B-GGUF ornith-1.0-9b-q4_k_m.gguf --local-dir "%MODELS_DIR%\deepreinforce-ai\Ornith-1.0-9B-GGUF"
if errorlevel 1 goto :error

:ds14
echo.
echo === 2/3: DeepSeek R1 Distill Qwen 14B (Q4_K_M) ===
mkdir "%MODELS_DIR%\lmstudio-community\DeepSeek-R1-Distill-Qwen-14B-GGUF" 2>nul
huggingface-cli download lmstudio-community/DeepSeek-R1-Distill-Qwen-14B-GGUF DeepSeek-R1-Distill-Qwen-14B-Q4_K_M.gguf --local-dir "%MODELS_DIR%\lmstudio-community\DeepSeek-R1-Distill-Qwen-14B-GGUF"
if errorlevel 1 goto :error

:ornith35
echo.
echo === 3/3: Ornith 1.0 35B (Q4_K_M) - High-End ===
mkdir "%MODELS_DIR%\deepreinforce-ai\Ornith-1.0-35B-GGUF" 2>nul
huggingface-cli download deepreinforce-ai/Ornith-1.0-35B-GGUF ornith-1.0-35b-q4_k_m.gguf --local-dir "%MODELS_DIR%\deepreinforce-ai\Ornith-1.0-35B-GGUF"
if errorlevel 1 goto :error

echo.
echo [go-database] ✓ Alle Modelle geladen nach: %MODELS_DIR%
echo Starte LM Studio neu, um die Modelle zu laden.
goto :eof

:error
echo [FEHLER] Download fehlgeschlagen. Stelle sicher, dass huggingface-cli installiert ist:
echo   pip install huggingface-hub
pause
exit /b 1
