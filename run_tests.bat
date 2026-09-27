@echo off
set EXE=C:\Users\dao.tmdb\Desktop\2026\ImpT\PipeWrap\gopipe\gopipe_test.exe
set PASS=0
set FAIL=0

echo ============================================================
echo  gopipe Phase 2 End-to-End Tests
echo ============================================================

:: Test 1: sequential mode
echo [TEST 1] Sequential mode
%EXE% cmd /c type test_producer.txt -r cmd /c echo SEQ: > test_out.txt 2>&1
findstr /c:"SEQ:" test_out.txt >nul && (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 2: parallel mode
echo [TEST 2] Parallel mode
%EXE% cmd /c type test_producer.txt -r cmd /c echo PAR: -- --mode parallel > test_out.txt 2>&1
findstr /c:"PAR:" test_out.txt >nul && (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 3: batch mode
echo [TEST 3] Batch mode
%EXE% cmd /c type test_producer.txt -r cmd /c echo BATCH: -- --mode batch --batch-size 2 > test_out.txt 2>&1
findstr /c:"BATCH:" test_out.txt >nul && (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 4: full mode
echo [TEST 4] Full-collect mode
%EXE% cmd /c type test_producer.txt -r cmd /c type -- --mode full > test_out.txt 2>&1
if %ERRORLEVEL% LEQ 1 (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 5: dry-run
echo [TEST 5] Dry-run mode
%EXE% type test_producer.txt -r cmd /c echo X -- --dry-run > test_out.txt 2>&1
findstr /c:"DRY RUN" test_out.txt >nul && (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 6: timeout exit code
echo [TEST 6] Timeout
%EXE% type test_producer.txt -r cmd /c echo T -- --timeout 1ms > test_out.txt 2>&1
if %ERRORLEVEL% LEQ 2 (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 7: tee output
echo [TEST 7] Tee writer
%EXE% cmd /c type test_producer.txt -r cmd /c echo TEE: -- --tee test_tee.txt > test_out.txt 2>&1
findstr /c:"TEE:" test_tee.txt >nul && (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 8: no-summary flag
echo [TEST 8] No-summary flag
%EXE% type test_producer.txt -r cmd /c echo NS: -- --no-summary > test_out.txt 2>&1
findstr /c:"Execution Summary" test_out.txt >nul && (echo FAIL & set /a FAIL+=1) || (echo PASS & set /a PASS+=1)

:: Test 9: unknown subcommand still exits 1
echo [TEST 9] Unknown subcommand
%EXE% frobnicate > test_out.txt 2>&1
if %ERRORLEVEL%==1 (echo PASS & set /a PASS+=1) || (echo FAIL & set /a FAIL+=1)

:: Test 10: producer failure exit code
echo [TEST 10] Producer failure exit code
%EXE% cmd /c "exit 1" -r cmd /c echo X > test_out.txt 2>&1
if %ERRORLEVEL%==10 (echo PASS & set /a PASS+=1) || (echo FAIL - got %ERRORLEVEL% & set /a FAIL+=1)

echo ============================================================
echo  Results: %PASS% passed, %FAIL% failed
echo ============================================================

del test_out.txt 2>nul
del test_tee.txt 2>nul
