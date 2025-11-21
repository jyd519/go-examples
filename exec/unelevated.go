package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type TokenElevation struct {
	TokenIsElevated uint32
}

// ProcessResult 进程执行结果结构体
type ProcessResult struct {
	Success   bool   // 执行是否成功
	ExitCode  uint32 // 进程退出代码
	StdOutput string // 标准输出
	StdError  string // 标准错误
	ProcessID uint32 // 进程ID
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: unelevated.exe <command>")
		fmt.Println("示例: unelevated.exe \"cmd.exe /c dir\"")
		fmt.Println("示例: unelevated.exe \"cmd.exe /c echo Hello World\"")
		fmt.Println("测试: unelevated.exe test")
		return
	}

	command := os.Args[1]

	// 检查当前是否以管理员权限运行
	if !isElevated() {
		fmt.Printf("当前进程没有管理员权限，直接运行命令: %s\n", command)
		// 直接运行命令并捕获输出
		result := runCommandWithOutput(command, 30*time.Second)
		printResult(result)
		return
	}

	fmt.Printf("当前进程有管理员权限，从explorer.exe复制token运行命令: %s\n", command)

	// 从explorer.exe复制token
	userToken, err := createUserToken()
	if err != nil {
		fmt.Printf("从explorer.exe复制token失败: %v\n", err)
		// 如果无法复制token，尝试直接运行
		fmt.Println("尝试直接运行命令...")
		result := runCommandWithOutput(command, 30*time.Second)
		printResult(result)
		return
	}
	defer userToken.Close()

	// 使用复制的token运行命令并捕获输出
	result := runProcessWithRestrictedTokenAndOutput(userToken, command, 30*time.Second, nil)
	printResult(result)
}

// runCommandWithOutput 直接运行命令并捕获输出
func runCommandWithOutput(command string, timeout time.Duration) ProcessResult {
	result := ProcessResult{Success: false}

	cmd := exec.Command("cmd", "/c", command)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// 启动进程
	if err := cmd.Start(); err != nil {
		result.StdError = fmt.Sprintf("启动进程失败: %v", err)
		return result
	}

	result.ProcessID = uint32(cmd.Process.Pid)
	fmt.Printf("进程启动成功，PID: %d\n", result.ProcessID)

	// 创建超时通道
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// 等待进程完成或超时
	select {
	case err := <-done:
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				result.ExitCode = uint32(exitError.ExitCode())
			}
			result.StdError += fmt.Sprintf("进程执行失败: %v", err)
		} else {
			result.ExitCode = 0
			result.Success = true
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		result.StdError = "进程执行超时"
		result.ExitCode = 1
	}

	result.StdOutput = stdoutBuf.String()
	if result.StdError == "" {
		result.StdError = stderrBuf.String()
	} else {
		result.StdError += "\n" + stderrBuf.String()
	}

	return result
}

// runProcessWithRestrictedTokenAndOutput 使用用户token运行进程并捕获输出
// customEnv: 自定义环境变量，如果为nil则使用默认环境变量
func runProcessWithRestrictedTokenAndOutput(token windows.Token, command string, timeout time.Duration, customEnv map[string]string) ProcessResult {
	result := ProcessResult{Success: false}

	// 创建用户环境块
	envBlock, err := createEnvironmentBlockWithCustomVars(token, customEnv)
	if err != nil {
		result.StdError = fmt.Sprintf("创建环境块失败: %v", err)
		return result
	}
	defer destroyEnvironmentBlock(envBlock)

	// 创建管道用于捕获输出
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		result.StdError = fmt.Sprintf("创建stdout管道失败: %v", err)
		return result
	}
	defer stdoutR.Close()
	defer stdoutW.Close()

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		result.StdError = fmt.Sprintf("创建stderr管道失败: %v", err)
		return result
	}
	defer stderrR.Close()
	defer stderrW.Close()

	// 配置进程启动参数
	si := &windows.StartupInfo{
		Cb:        uint32(unsafe.Sizeof(windows.StartupInfo{})),
		Flags:     windows.STARTF_USESTDHANDLES | windows.STARTF_USESHOWWINDOW,
		StdOutput: windows.Handle(stdoutW.Fd()),
		StdErr:    windows.Handle(stderrW.Fd()),
	}
	pi := &windows.ProcessInformation{}

	// 准备命令行
	cmdLine := fmt.Sprintf("cmd /c %s", command)

	// 使用CreateProcessWithTokenW API
	kernel32 := windows.NewLazySystemDLL("Advapi32.dll")
	procCreateProcessWithTokenW := kernel32.NewProc("CreateProcessWithTokenW")

	// 转换命令行为UTF16
	cmdLinePtr, err := windows.UTF16FromString(cmdLine)
	if err != nil {
		result.StdError = fmt.Sprintf("转换命令行失败: %v", err)
		return result
	}

	dir, err := filepath.Abs(".")
	if err != nil {
		result.StdError = fmt.Sprintf("获取当前目录失败: %v", err)
		return result
	}
	dirPtr, err := windows.UTF16FromString(dir)

	// 调用CreateProcessWithTokenW
	ret, _, err := procCreateProcessWithTokenW.Call(
		uintptr(token),
		1, // LOGON_WITH_PROFILE
		0, // lpApplicationName
		uintptr(unsafe.Pointer(&cmdLinePtr[0])),
		windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW,
		envBlock,
		uintptr(unsafe.Pointer(&dirPtr[0])),
		uintptr(unsafe.Pointer(si)),
		uintptr(unsafe.Pointer(pi)),
	)

	if ret == 0 {
		result.StdError = fmt.Sprintf("CreateProcessWithTokenW失败: %v", err)
		return result
	}

	result.ProcessID = pi.ProcessId
	fmt.Printf("受限权限进程启动成功，PID: %d\n", result.ProcessID)

	// 关闭写入端，这样管道在子进程结束时会关闭
	stdoutW.Close()
	stderrW.Close()

	// 创建goroutine读取输出
	stdoutChan := make(chan string)
	stderrChan := make(chan string)

	go func() {
		output, _ := io.ReadAll(stdoutR)
		stdoutChan <- convertFromGBK(output)
	}()

	go func() {
		output, _ := io.ReadAll(stderrR)
		stderrChan <- convertFromGBK(output)
	}()

	// 等待进程完成或超时
	waitResult, err := windows.WaitForSingleObject(pi.Process, uint32(timeout.Milliseconds()))
	if err != nil {
		result.StdError = fmt.Sprintf("等待进程失败: %v", err)
		windows.TerminateProcess(pi.Process, 1)
	} else if waitResult == windows.WAIT_OBJECT_0 {
		// 进程正常结束
		err = windows.GetExitCodeProcess(pi.Process, &result.ExitCode)
		if err != nil {
			result.StdError = fmt.Sprintf("获取退出代码失败: %v", err)
		} else {
			result.Success = true
			fmt.Printf("进程完成，退出代码: %d\n", result.ExitCode)
		}
	} else if waitResult == uint32(windows.WAIT_TIMEOUT) {
		fmt.Println("进程超时，正在终止...")
		windows.TerminateProcess(pi.Process, 1)
		result.StdError = "进程执行超时"
		result.ExitCode = 1
	} else {
		result.StdError = fmt.Sprintf("等待进程失败，返回值: %d", waitResult)
		windows.TerminateProcess(pi.Process, 1)
	}

	// 读取输出
	select {
	case result.StdOutput = <-stdoutChan:
	case <-time.After(5 * time.Second):
		result.StdOutput = "读取stdout超时"
	}

	select {
	case stderrOutput := <-stderrChan:
		if result.StdError == "" {
			result.StdError = stderrOutput
		} else if stderrOutput != "" {
			result.StdError += "\n" + stderrOutput
		}
	case <-time.After(5 * time.Second):
		if result.StdError == "" {
			result.StdError = "读取stderr超时"
		}
	}

	// 关闭句柄
	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)

	return result
}

// printResult 打印执行结果
func printResult(result ProcessResult) {
	fmt.Printf("\n=== 执行结果 ===\n")
	fmt.Printf("成功: %t\n", result.Success)
	fmt.Printf("进程ID: %d\n", result.ProcessID)
	fmt.Printf("退出代码: %d\n", result.ExitCode)

	if result.StdOutput != "" {
		fmt.Printf("\n标准输出:\n%s\n", result.StdOutput)
	}

	if result.StdError != "" {
		fmt.Printf("\n标准错误:\n%s\n", result.StdError)
	}
	fmt.Printf("================\n")
}

func isElevated() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation TokenElevation
	var returnedLen uint32
	err = windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &returnedLen)
	if err != nil {
		return false
	}

	return elevation.TokenIsElevated != 0
}

func enablePrivilege(privilege string) error {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	defer token.Close()

	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, windows.StringToUTF16Ptr(privilege), &luid)
	if err != nil {
		return err
	}

	privileges := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{
				Luid:       luid,
				Attributes: windows.SE_PRIVILEGE_ENABLED,
			},
		},
	}

	return windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil)
}

// createUserToken 从explorer.exe复制令牌
func createUserToken() (windows.Token, error) {
	// 1. 查找 Explorer.exe 的 PID
	explorerPID, err := findExplorerPID()
	if err != nil {
		return 0, fmt.Errorf("查找explorer.exe失败: %v", err)
	}

	fmt.Printf("找到explorer.exe PID: %d\n", explorerPID)

	// 2. 获取 Explorer.exe 的令牌
	explorerToken, err := getExplorerToken(explorerPID)
	if err != nil {
		return 0, fmt.Errorf("获取explorer.exe令牌失败: %v", err)
	}
	defer explorerToken.Close()

	// 3. 复制令牌（用于创建新进程）
	var newToken windows.Token
	err = windows.DuplicateTokenEx(
		explorerToken,
		windows.TOKEN_ALL_ACCESS,
		nil,
		windows.SecurityIdentification,
		windows.TokenPrimary,
		&newToken,
	)
	if err != nil {
		return 0, fmt.Errorf("复制explorer.exe令牌失败: %v", err)
	}

	fmt.Println("成功复制explorer.exe令牌")
	return newToken, nil
}

// findExplorerPID 查找explorer.exe的进程ID
func findExplorerPID() (uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)

	var pe32 windows.ProcessEntry32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	err = windows.Process32First(snapshot, &pe32)
	if err != nil {
		return 0, err
	}

	for {
		exeName := windows.UTF16ToString(pe32.ExeFile[:])
		if exeName == "explorer.exe" {
			return pe32.ProcessID, nil
		}

		err = windows.Process32Next(snapshot, &pe32)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return 0, err
		}
	}

	return 0, fmt.Errorf("未找到explorer.exe进程")
}

// getExplorerToken 获取explorer.exe的令牌
func getExplorerToken(pid uint32) (windows.Token, error) {
	// 打开explorer.exe进程
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return 0, fmt.Errorf("打开explorer.exe进程失败: %v", err)
	}
	defer windows.CloseHandle(hProcess)

	// 获取进程令牌
	var token windows.Token
	err = windows.OpenProcessToken(hProcess, windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &token)
	if err != nil {
		return 0, fmt.Errorf("获取explorer.exe令牌失败: %v", err)
	}

	return token, nil
}

// createEnvironmentBlock 创建用户环境块
func createEnvironmentBlock(token windows.Token) (uintptr, error) {
	var envBlock uintptr

	// 调用CreateEnvironmentBlock
	userenv := windows.NewLazySystemDLL("userenv.dll")
	procCreateEnvironmentBlock := userenv.NewProc("CreateEnvironmentBlock")

	ret, _, err := procCreateEnvironmentBlock.Call(
		uintptr(unsafe.Pointer(&envBlock)),
		uintptr(token),
		0, // FALSE
	)

	if ret == 0 {
		return 0, fmt.Errorf("CreateEnvironmentBlock失败: %v", err)
	}

	return envBlock, nil
}

// destroyEnvironmentBlock 销毁环境块
func destroyEnvironmentBlock(envBlock uintptr) {
	if envBlock != 0 {
		userenv := windows.NewLazySystemDLL("userenv.dll")
		procDestroyEnvironmentBlock := userenv.NewProc("DestroyEnvironmentBlock")
		procDestroyEnvironmentBlock.Call(envBlock)
	}
}

// createEnvironmentBlockWithCustomVars 创建用户环境块并添加自定义环境变量
func createEnvironmentBlockWithCustomVars(token windows.Token, customEnv map[string]string) (uintptr, error) {
	// 首先创建基础环境块
	baseEnvBlock, err := createEnvironmentBlock(token)
	if err != nil {
		return 0, err
	}

	// 如果没有自定义环境变量，直接返回基础环境块
	if len(customEnv) == 0 {
		return baseEnvBlock, nil
	}

	// 销毁基础环境块，我们将创建新的
	defer destroyEnvironmentBlock(baseEnvBlock)

	// 获取当前环境变量
	envStrings := os.Environ()
	envMap := make(map[string]string)

	// 解析现有环境变量
	for _, env := range envStrings {
		if idx := strings.Index(env, "="); idx != -1 {
			key := env[:idx]
			value := env[idx+1:]
			envMap[key] = value
		}
	}

	// 添加自定义环境变量
	for key, value := range customEnv {
		envMap[key] = value
	}

	// 构建环境块
	var envBlock []uint16
	for key, value := range envMap {
		envStr := key + "=" + value
		b, _ := windows.UTF16FromString(envStr)
		envBlock = append(envBlock, b...)
	}
	// 添加终止符
	envBlock = append(envBlock, 0)

	// 分配内存并复制数据
	size := len(envBlock) * 2 // 每个uint16占2字节
	ptr, err := windows.LocalAlloc(windows.LMEM_FIXED, uint32(size))
	if err != nil {
		return 0, fmt.Errorf("分配环境块内存失败: %v", err)
	}
	if ptr == 0 {
		return 0, fmt.Errorf("分配环境块内存失败")
	}

	// 复制数据到分配的内存
	copy((*[1 << 20]uint16)(unsafe.Pointer(ptr))[:len(envBlock)], envBlock)

	return ptr, nil
}

// convertFromGBK 将GBK编码的字节转换为UTF-8字符串
func convertFromGBK(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 尝试检测是否为有效的UTF-8
	if isValidUTF8(data) {
		return string(data)
	}

	// 如果不是UTF-8，尝试从GBK转换
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Data, _, err := transform.Bytes(decoder, data)
	if err != nil {
		// 如果转换失败，返回原始字符串
		return string(data)
	}

	return string(utf8Data)
}

// isValidUTF8 检查字节序列是否为有效的UTF-8
func isValidUTF8(data []byte) bool {
	for len(data) > 0 {
		r, size := decodeUTF8Rune(data)
		if r == 0xFFFD && size == 1 {
			return false
		}
		data = data[size:]
	}
	return true
}

// decodeUTF8Rune 解码UTF-8字符
func decodeUTF8Rune(data []byte) (rune, int) {
	if len(data) == 0 {
		return 0, 0
	}

	b0 := data[0]
	if b0 < 0x80 {
		return rune(b0), 1
	}

	if b0 < 0xC0 {
		return 0xFFFD, 1
	}

	if len(data) < 2 {
		return 0xFFFD, 1
	}

	b1 := data[1]
	if b0 < 0xE0 {
		if b1&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		r := rune(b0&0x1F)<<6 | rune(b1&0x3F)
		if r < 0x80 {
			return 0xFFFD, 1
		}
		return r, 2
	}

	if len(data) < 3 {
		return 0xFFFD, 1
	}

	b2 := data[2]
	if b0 < 0xF0 {
		if b1&0xC0 != 0x80 || b2&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		r := rune(b0&0x0F)<<12 | rune(b1&0x3F)<<6 | rune(b2&0x3F)
		if r < 0x800 {
			return 0xFFFD, 1
		}
		return r, 3
	}

	if len(data) < 4 {
		return 0xFFFD, 1
	}

	b3 := data[3]
	if b0 < 0xF8 {
		if b1&0xC0 != 0x80 || b2&0xC0 != 0x80 || b3&0xC0 != 0x80 {
			return 0xFFFD, 1
		}
		r := rune(b0&0x07)<<18 | rune(b1&0x3F)<<12 | rune(b2&0x3F)<<6 | rune(b3&0x3F)
		if r < 0x10000 || r > 0x10FFFF {
			return 0xFFFD, 1
		}
		return r, 4
	}

	return 0xFFFD, 1
}
