#include <Windows.h>
#include <TlHelp32.h>
#include <userenv.h>
#include <string>
#include <vector>
#include <iostream>

#pragma comment(lib, "userenv.lib")

// 进程执行结果结构体
struct ProcessResult {
    bool success;
    DWORD exitCode;
    std::string stdOutput;
    std::string stdError;
};

// 从管道读取数据的辅助函数
std::string ReadFromPipe(HANDLE hPipe) {
    std::string result;
    char buffer[4096];
    DWORD bytesRead;

    while (true) {
        if (!ReadFile(hPipe, buffer, sizeof(buffer) - 1, &bytesRead, nullptr)) {
            DWORD error = GetLastError();
            if (error == ERROR_BROKEN_PIPE) {
                break; // 管道已关闭，正常结束
            }
            break;
        }

        if (bytesRead == 0) {
            break;
        }

        buffer[bytesRead] = '\0';
        result += buffer;
    }

    return result;
}

// 创建安全的管道
bool CreateSecurePipe(HANDLE* hReadPipe, HANDLE* hWritePipe, bool inheritWrite = true) {
    SECURITY_ATTRIBUTES sa;
    sa.nLength = sizeof(SECURITY_ATTRIBUTES);
    sa.bInheritHandle = TRUE;
    sa.lpSecurityDescriptor = nullptr;

    if (!CreatePipe(hReadPipe, hWritePipe, &sa, 0)) {
        return false;
    }

    // 确保读取端不被子进程继承
    if (!SetHandleInformation(*hReadPipe, HANDLE_FLAG_INHERIT, 0)) {
        CloseHandle(*hReadPipe);
        CloseHandle(*hWritePipe);
        return false;
    }

    return true;
}

ProcessResult RunAsCurrentUserWithOutput(LPCWSTR lpApplicationPath, LPCWSTR lpCommandLine = nullptr, DWORD timeoutMs = 30000) {
    ProcessResult result = {false, 0, "", ""};

    // 1. 查找 Explorer.exe 的 PID
    DWORD dwExplorerPID = 0;
    HANDLE hSnapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if (hSnapshot == INVALID_HANDLE_VALUE) {
        std::wcerr << L"Failed to create process snapshot" << std::endl;
        return result;
    }

    PROCESSENTRY32W pe32 = { sizeof(pe32) };
    if (Process32FirstW(hSnapshot, &pe32)) {
        do {
            if (_wcsicmp(pe32.szExeFile, L"explorer.exe") == 0) {
                dwExplorerPID = pe32.th32ProcessID;
                break;
            }
        } while (Process32NextW(hSnapshot, &pe32));
    }
    CloseHandle(hSnapshot);

    if (dwExplorerPID == 0) {
        std::wcerr << L"Explorer.exe not found" << std::endl;
        return result;
    }

    // 2. 获取 Explorer.exe 的令牌
    HANDLE hExplorerProcess = OpenProcess(PROCESS_QUERY_INFORMATION, FALSE, dwExplorerPID);
    if (!hExplorerProcess) {
        std::wcerr << L"Failed to open explorer process, error: " << GetLastError() << std::endl;
        return result;
    }

    HANDLE hToken = nullptr;
    if (!OpenProcessToken(hExplorerProcess, TOKEN_DUPLICATE | TOKEN_QUERY, &hToken)) {
        std::wcerr << L"Failed to open process token, error: " << GetLastError() << std::endl;
        CloseHandle(hExplorerProcess);
        return result;
    }
    CloseHandle(hExplorerProcess);

    // 3. 复制令牌（用于创建新进程）
    HANDLE hNewToken = nullptr;
    if (!DuplicateTokenEx(hToken, TOKEN_ALL_ACCESS, nullptr, SecurityIdentification, TokenPrimary, &hNewToken)) {
        std::wcerr << L"Failed to duplicate token, error: " << GetLastError() << std::endl;
        CloseHandle(hToken);
        return result;
    }
    CloseHandle(hToken);

    // 4. 创建进程环境块
    LPVOID lpEnvironment = nullptr;
    if (!CreateEnvironmentBlock(&lpEnvironment, hNewToken, FALSE)) {
        std::wcerr << L"Failed to create environment block, error: " << GetLastError() << std::endl;
        CloseHandle(hNewToken);
        return result;
    }

    // 5. 创建管道用于捕获输出
    HANDLE hStdOutRead = nullptr, hStdOutWrite = nullptr;
    HANDLE hStdErrRead = nullptr, hStdErrWrite = nullptr;

    if (!CreateSecurePipe(&hStdOutRead, &hStdOutWrite)) {
        std::wcerr << L"Failed to create stdout pipe, error: " << GetLastError() << std::endl;
        if (lpEnvironment) DestroyEnvironmentBlock(lpEnvironment);
        CloseHandle(hNewToken);
        return result;
    }

    if (!CreateSecurePipe(&hStdErrRead, &hStdErrWrite)) {
        std::wcerr << L"Failed to create stderr pipe, error: " << GetLastError() << std::endl;
        CloseHandle(hStdOutRead);
        CloseHandle(hStdOutWrite);
        if (lpEnvironment) DestroyEnvironmentBlock(lpEnvironment);
        CloseHandle(hNewToken);
        return result;
    }

    // 6. 设置启动信息
    STARTUPINFOW si = { sizeof(si) };
    si.dwFlags = STARTF_USESTDHANDLES | STARTF_USESHOWWINDOW;
    si.wShowWindow = SW_HIDE; // 隐藏窗口
    si.hStdOutput = hStdOutWrite;
    si.hStdError = hStdErrWrite;
    si.hStdInput = GetStdHandle(STD_INPUT_HANDLE);

    // 7. 准备命令行
    std::wstring commandLine;
    if (lpCommandLine && wcslen(lpCommandLine) > 0) {
        commandLine = lpCommandLine;
    } else {
        commandLine = lpApplicationPath;
    }

    // CreateProcessWithTokenW需要可修改的字符串
    std::vector<wchar_t> cmdBuffer(commandLine.begin(), commandLine.end());
    cmdBuffer.push_back(L'\0');

    // 8. 启动目标进程
    PROCESS_INFORMATION pi = {};
    BOOL bSuccess = CreateProcessWithTokenW(
        hNewToken,
        LOGON_WITH_PROFILE,
        lpApplicationPath,
        cmdBuffer.data(),
        CREATE_UNICODE_ENVIRONMENT | CREATE_NO_WINDOW,
        lpEnvironment,
        nullptr,
        &si,
        &pi
    );

    // 9. 关闭写入端句柄（重要：这样子进程结束时管道会关闭）
    CloseHandle(hStdOutWrite);
    CloseHandle(hStdErrWrite);

    if (bSuccess) {
        std::wcout << L"Process started successfully, PID: " << pi.dwProcessId << std::endl;

        // 10. 等待进程完成并读取输出
        DWORD waitResult = WaitForSingleObject(pi.hProcess, timeoutMs);

        if (waitResult == WAIT_OBJECT_0) {
            // 进程正常结束，获取退出代码
            GetExitCodeProcess(pi.hProcess, &result.exitCode);
            result.success = true;

            // 读取标准输出和标准错误
            result.stdOutput = ReadFromPipe(hStdOutRead);
            result.stdError = ReadFromPipe(hStdErrRead);

            std::wcout << L"Process completed with exit code: " << result.exitCode << std::endl;
        } else if (waitResult == WAIT_TIMEOUT) {
            std::wcerr << L"Process timed out, terminating..." << std::endl;
            TerminateProcess(pi.hProcess, 1);
            result.stdError = "Process timed out";
        } else {
            std::wcerr << L"Wait failed, error: " << GetLastError() << std::endl;
            result.stdError = "Wait failed";
        }

        CloseHandle(pi.hProcess);
        CloseHandle(pi.hThread);
    } else {
        std::wcerr << L"Failed to create process, error: " << GetLastError() << std::endl;
    }

    // 11. 清理资源
    CloseHandle(hStdOutRead);
    CloseHandle(hStdErrRead);
    if (lpEnvironment) DestroyEnvironmentBlock(lpEnvironment);
    CloseHandle(hNewToken);

    return result;
}

// 简化版本，兼容原有接口
bool RunAsCurrentUser(LPCWSTR lpApplicationPath) {
    ProcessResult result = RunAsCurrentUserWithOutput(lpApplicationPath);

    if (!result.stdOutput.empty()) {
        std::cout << "STDOUT:\n" << result.stdOutput << std::endl;
    }

    if (!result.stdError.empty()) {
        std::cout << "STDERR:\n" << result.stdError << std::endl;
    }

    return result.success;
}

// 测试函数
void TestRunAsCurrentUser() {
    std::wcout << L"Testing RunAsCurrentUserWithOutput..." << std::endl;

    // 测试1: 运行cmd命令
    std::wcout << L"\n=== Test 1: Running 'cmd /c echo Hello World' ===" << std::endl;
    ProcessResult result1 = RunAsCurrentUserWithOutput(L"C:\\Windows\\System32\\cmd.exe", L"cmd /c echo Hello World");

    std::wcout << L"Success: " << (result1.success ? L"Yes" : L"No") << std::endl;
    std::wcout << L"Exit Code: " << result1.exitCode << std::endl;
    if (!result1.stdOutput.empty()) {
        std::cout << "STDOUT: " << result1.stdOutput << std::endl;
    }
    if (!result1.stdError.empty()) {
        std::cout << "STDERR: " << result1.stdError << std::endl;
    }

    // 测试2: 运行dir命令
    std::wcout << L"\n=== Test 2: Running 'cmd /c dir C:\\' ===" << std::endl;
    ProcessResult result2 = RunAsCurrentUserWithOutput(L"C:\\Windows\\System32\\cmd.exe", L"cmd /c dir C:\\");

    std::wcout << L"Success: " << (result2.success ? L"Yes" : L"No") << std::endl;
    std::wcout << L"Exit Code: " << result2.exitCode << std::endl;
    if (!result2.stdOutput.empty()) {
        std::cout << "STDOUT (first 500 chars): " << result2.stdOutput.substr(0, 500) << "..." << std::endl;
    }

    // 测试3: 运行一个会产生错误的命令
    std::wcout << L"\n=== Test 3: Running 'cmd /c dir NonExistentFolder' ===" << std::endl;
    ProcessResult result3 = RunAsCurrentUserWithOutput(L"C:\\Windows\\System32\\cmd.exe", L"cmd /c dir NonExistentFolder");

    std::wcout << L"Success: " << (result3.success ? L"Yes" : L"No") << std::endl;
    std::wcout << L"Exit Code: " << result3.exitCode << std::endl;
    if (!result3.stdOutput.empty()) {
        std::cout << "STDOUT: " << result3.stdOutput << std::endl;
    }
    if (!result3.stdError.empty()) {
        std::cout << "STDERR: " << result3.stdError << std::endl;
    }

    // 测试4: 运行notepad（测试GUI应用）
    std::wcout << L"\n=== Test 4: Running 'notepad.exe' ===" << std::endl;
    ProcessResult result4 = RunAsCurrentUserWithOutput(L"C:\\Windows\\System32\\notepad.exe", nullptr, 3000); // 3秒超时

    std::wcout << L"Success: " << (result4.success ? L"Yes" : L"No") << std::endl;
    std::wcout << L"Exit Code: " << result4.exitCode << std::endl;
}

int main() {
    // 设置控制台输出编码
    SetConsoleOutputCP(CP_UTF8);

    std::wcout << L"Windows进程管理器 - 以当前用户权限运行并捕获输出" << std::endl;
    std::wcout << L"================================================" << std::endl;

    // 运行测试
    TestRunAsCurrentUser();

    return 0;
}