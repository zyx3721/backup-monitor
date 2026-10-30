@echo off
:: ============================================================================
:: Veeam Endpoint Backup - 执行备份脚本
:: 作者：Jerion
:: 日期：2026-10-29
:: 描述：此脚本用于关闭相关 Veeam 服务及进程后，重新启动服务并执行一次备份任务。
:: ============================================================================

title Veeam 备份任务执行脚本

set "statusFile=c:\backup-monitor\veeam-status.txt"

for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set "dt=%%I"
set "time=%dt:~0,4%-%dt:~4,2%-%dt:~6,2% %dt:~8,2%:%dt:~10,2%:%dt:~12,2%"
> "%statusFile%" echo start=%time%

:: 关闭 Veeam 相关进程
echo [信息] 正在终止 Veeam 相关进程...
taskkill /f /im Veeam.EndPoint.Service.exe >nul 2>&1
taskkill /f /im Veeam.EndPoint.Manager.exe >nul 2>&1
taskkill /f /im VeeamAgent.exe >nul 2>&1

:: 启动 Veeam Endpoint Backup 服务
echo [信息] 启动 Veeam Endpoint Backup 服务...
net start VeeamEndpointBackupSvc >nul 2>&1

:: 强制终止 Volume Shadow Copy 服务
echo [警告] 终止系统 VSS 服务（vssvc.exe）...
taskkill /f /im vssvc.exe >nul 2>&1

:: 执行指定的备份任务，任选其一，注意更新C:\ProgramData\Veeam\EndpointData
echo [信息] 开始执行 E盘 备份任务...
"C:\Program Files\Veeam\Endpoint Backup\Veeam.EndPoint.Manager.exe" "backup" "d5cfede2-0cce-44db-bead-56cd57aa7410" 
echo [完成] 备份命令已执行

for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set "dt=%%I"
set "time=%dt:~0,4%-%dt:~4,2%-%dt:~6,2% %dt:~8,2%:%dt:~10,2%:%dt:~12,2%"
>> "%statusFile%" echo end=%time%

ping -n 6 127.1 >nul
exit