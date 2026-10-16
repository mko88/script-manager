@echo off
REM Builds the solution.
REM
REM Usage: build.bat [/clean] <config> [platform]
REM
REM Options:
REM   /clean               Remove old output first
REM   /v LEVEL             Verbosity {quiet|normal|detailed}
:: --out DIR             Output folder
setlocal
msbuild
