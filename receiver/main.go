package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const port = "9999"

const (
	cGreen  = "\033[92m"
	cBlue   = "\033[94m"
	cCyan   = "\033[96m"
	cRed    = "\033[91m"
	cYellow = "\033[93m"
	cWhite  = "\033[97m"
	cDim    = "\033[90m"
	cReset  = "\033[0m"
)

// logMu protects console output from interleaving across concurrent goroutines.
var logMu sync.Mutex

func logf(format string, args ...interface{}) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Printf(format, args...)
}

// enableANSI activates VT100 escape code processing on the Windows console.
func enableANSI() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	getMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	setMode.Call(uintptr(handle), uintptr(mode|0x0004))
}

func showBanner() {
	fmt.Print("\033[2J\033[H")
	fmt.Print(cGreen)
	fmt.Println("   ____  _           _   _                  _    ____")
	fmt.Println("  |  _ \\| |__   __ _| \\ | |_ ___  _ __ ___ / \\  |  _ \\")
	fmt.Println("  | |_) | '_ \\ / _` |  \\| | __/ _ \\| '_ ` _ \\ / _ \\ | |_) |")
	fmt.Println("  |  __/| | | | (_| | |\\  | || (_) | | | | | / ___ \\|  __/")
	fmt.Println("  |_|   |_| |_|\\__,_|_| \\_|\\__\\___/|_| |_| /_/   \\_\\_|")
	fmt.Print(cReset)
	fmt.Println(cBlue + "        Built Miracle Gideon" + cReset)
	fmt.Println()
}

// getLocalIPs returns all active non-loopback IPv4 addresses with their interface names.
func getLocalIPs() []string {
	var result []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"UNKNOWN"}
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			result = append(result, fmt.Sprintf("%-18s [%s]", ip.String(), iface.Name))
		}
	}
	if len(result) == 0 {
		return []string{"UNKNOWN"}
	}
	return result
}

// runCmd executes a command and returns combined stdout+stderr as a string.
func runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("ERR :: %v\n%s", err, string(out))
	}
	return string(out)
}

// runPS executes a PowerShell script, forcing UTF-8 output encoding.
func runPS(script string) string {
	full := "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " + script
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", full)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("ERR :: %v\n%s", err, string(out))
	}
	return string(out)
}

// executeModule dispatches a command string to its corresponding Windows function.
func executeModule(command, arg string) string {
	switch command {

	case "PROBE_NEARBY":
		return runCmd("netsh", "wlan", "show", "networks", "mode=bssid")

	case "WIFI_PASS_ALL":
		// Select-String returns MatchInfo objects — use .Line to get the raw string.
		// Select-Object -First 1 on $k guards against multiple 'Key Content' matches
		// returning a collection instead of a single object, which breaks .Line access.
		return runPS(`
			$profiles = (netsh wlan show profiles) |
				Select-String 'All User Profile' |
				ForEach-Object { ($_.Line -split ':\s+', 2)[1].Trim() }
			if (-not $profiles) {
				"INFO :: NO SAVED WLAN PROFILES FOUND"
				return
			}
			foreach ($p in $profiles) {
				$d = netsh wlan show profile name="$p" key=clear
				$k = $d | Select-String 'Key Content' | Select-Object -First 1
				"PROFILE : $p"
				if ($k) { "KEY     : $(($k.Line -split ':\s+', 2)[1].Trim())" }
				else    { "KEY     : [OPEN NETWORK OR NO KEY STORED]" }
				""
			}
		`)

	case "WIFI_PASS_TARGET":
		if arg == "" {
			return "ERR :: ARGUMENT REQUIRED\nUSAGE   :: WIFI_PASS_TARGET:<SSID_NAME>\n"
		}
		// FIX: Use PowerShell to properly quote the SSID name.
		// runCmd cannot quote arguments — SSIDs with spaces silently fail via netsh directly.
		return runPS(fmt.Sprintf(`netsh wlan show profile name="%s" key=clear`, arg))

	case "PROF_LIST":
		return runCmd("netsh", "wlan", "show", "profiles")

	case "ARP_SWEEP":
		return runCmd("arp", "-a")

	case "PORT_AUDIT":
		return runPS("netstat -ano | Select-String 'LISTENING'")

	case "DNS_TRACE":
		dns := runPS("ipconfig /all | Select-String 'DNS'")
		lookup := runCmd("nslookup", "google.com")
		return dns + "\n--- NSLOOKUP ---\n" + lookup

	case "CONN_MONITOR":
		return runPS("netstat -an | Select-String 'ESTABLISHED'")

	case "DUMP_REPORT":
		sections := []string{
			"SYS_ENUM", "ADAPTER_DEEP", "WIFI_PASS_ALL",
			"ARP_SWEEP", "PORT_AUDIT", "CONN_MONITOR",
			"ROUTE_TABLE", "DISK_INFO", "PROC_ENUM",
		}
		var sb strings.Builder
		for _, m := range sections {
			sb.WriteString(fmt.Sprintf("\n=== %s ===\n", m))
			sb.WriteString(executeModule(m, ""))
		}
		return sb.String()

	case "SYS_ENUM":
		return runPS(`
			$os = Get-CimInstance Win32_OperatingSystem
			$cs = Get-CimInstance Win32_ComputerSystem
			"HOSTNAME  : " + $env:COMPUTERNAME
			"USER      : " + $env:USERNAME
			"DOMAIN    : " + $cs.Domain
			"OS        : " + $os.Caption
			"BUILD     : " + $os.BuildNumber
			"ARCH      : " + $cs.SystemType
			"UPTIME    : " + ((Get-Date) - $os.LastBootUpTime).ToString()
			"INSTALL   : " + $os.InstallDate
		`)

	case "PROC_ENUM":
		return runCmd("tasklist", "/fo", "table")

	case "ROUTE_TABLE":
		return runCmd("route", "print")

	case "FIREWALL_AUDIT":
		return runCmd("netsh", "advfirewall", "show", "allprofiles")

	case "SHARES_ENUM":
		a := runCmd("net", "share")
		b := runCmd("net", "use")
		return a + "\n--- MAPPED DRIVES ---\n" + b

	case "USER_ENUM":
		a := runCmd("net", "user")
		b := runCmd("net", "localgroup", "administrators")
		return a + "\n--- ADMINISTRATOR GROUP ---\n" + b

	case "STARTUP_ENUM":
		return runPS(`
			Get-CimInstance Win32_StartupCommand |
				Select-Object Name, Command, Location, User |
				Format-List
		`)

	case "CRED_VAULT":
		return runCmd("cmdkey", "/list")

	case "RDP_STATUS":
		// FIX: Added -ErrorAction SilentlyContinue and null check.
		// Without it, the command crashes if the key is inaccessible (non-admin or Home edition).
		return runPS(`
			$key = Get-ItemProperty 'HKLM:\System\CurrentControlSet\Control\Terminal Server' -ErrorAction SilentlyContinue
			if ($null -eq $key) {
				"RDP STATUS : REGISTRY KEY NOT ACCESSIBLE"
			} elseif ($key.fDenyTSConnections -eq 0) {
				"RDP STATUS : ENABLED"
			} else {
				"RDP STATUS : DISABLED"
			}
			""
			try { query session 2>&1 } catch { "SESSION QUERY : NOT AVAILABLE ON THIS WINDOWS EDITION" }
		`)

	case "PROXY_CFG":
		// FIX: Added -ErrorAction SilentlyContinue and null check.
		// Without it, the command throws if proxy has never been configured (key properties missing).
		reg := runPS(`
			$p = Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -ErrorAction SilentlyContinue
			if ($p) {
				"PROXY ENABLE  : " + $p.ProxyEnable
				"PROXY SERVER  : " + $p.ProxyServer
				"PROXY BYPASS  : " + $p.ProxyOverride
			} else {
				"INFO :: No proxy registry settings found"
			}
		`)
		winhttp := runCmd("netsh", "winhttp", "show", "proxy")
		return reg + "\n--- WINHTTP ---\n" + winhttp

	case "HOSTS_READ":
		return runPS(`Get-Content "$env:SystemRoot\System32\drivers\etc\hosts"`)

	case "ADAPTER_DEEP":
		return runCmd("ipconfig", "/all")

	case "BT_SCAN":
		// FIX: Added empty result check — returns silent empty table otherwise.
		return runPS(`
			$devices = Get-PnpDevice -ErrorAction SilentlyContinue |
				Where-Object { $_.Class -eq 'Bluetooth' }
			if ($devices) {
				$devices | Select-Object FriendlyName, Status, DeviceID | Format-Table -AutoSize
			} else {
				"INFO :: NO BLUETOOTH DEVICES FOUND"
			}
		`)

	case "ENV_DUMP":
		return runCmd("cmd", "/c", "set")

	case "DISK_INFO":
		// FIX: Added null guards on Used/Free.
		// Network drives and virtual drives have null for these fields —
		// arithmetic on null throws a terminating error in PowerShell.
		return runPS(`
			$drives = Get-PSDrive -PSProvider FileSystem -ErrorAction SilentlyContinue
			if ($drives) {
				$drives | Select-Object Name,
					@{N='Used_GB';E={ if ($_.Used)  { [math]::Round($_.Used/1GB, 2)  } else { 'N/A' } }},
					@{N='Free_GB';E={ if ($_.Free)  { [math]::Round($_.Free/1GB, 2)  } else { 'N/A' } }},
					@{N='Total_GB';E={ if ($_.Used -and $_.Free) { [math]::Round(($_.Used+$_.Free)/1GB, 2) } else { 'N/A' } }} |
				Format-Table -AutoSize
			}
			""
			Get-Volume -ErrorAction SilentlyContinue | Format-Table -AutoSize
		`)

	default:
		return "ERR :: UNKNOWN MODULE :: " + command + "\n"
	}
}

// handleConn reads one command, executes the module, and writes the response.
func handleConn(conn net.Conn) {
	defer conn.Close()

	// FIX: Read the command first with a short initial timeout,
	// THEN set the execution deadline based on which module was requested.
	// Previous version set a flat 120s deadline before reading — DUMP_REPORT
	// runs 9 modules sequentially and reliably exceeded it.
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	reader := bufio.NewReader(conn)
	raw, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, ":", 2)
	command := strings.ToUpper(strings.TrimSpace(parts[0]))
	arg := ""
	if len(parts) == 2 {
		arg = strings.TrimSpace(parts[1])
	}

	// Give DUMP_REPORT extra time — it runs 9 modules sequentially.
	deadline := 120 * time.Second
	if command == "DUMP_REPORT" {
		deadline = 300 * time.Second
	}
	conn.SetDeadline(time.Now().Add(deadline))

	// FIX: Use logf (mutex-protected) instead of fmt.Printf.
	// handleConn runs in a goroutine — concurrent connections interleave output without a lock.
	logf("%s[>>]%s %s%s%s\n", cCyan, cReset, cWhite, command, cReset)

	output := executeModule(command, arg)

	writer := bufio.NewWriter(conn)
	writer.WriteString(output)
	if !strings.HasSuffix(output, "\n") {
		writer.WriteString("\n")
	}
	writer.WriteString("<<END>>\n")
	writer.Flush()

	logf("%s[OK]%s %s\n", cGreen, cReset, command)
}

func main() {
	enableANSI()
	showBanner()

	ips := getLocalIPs()
	fmt.Printf("%sMODE%s     :: RECEIVER\n", cCyan, cReset)
	fmt.Printf("%sPORT%s     :: %s%s%s\n", cDim, cReset, cYellow, port, cReset)
	fmt.Println()
	fmt.Printf("%sLOCAL ADDRESSES :%s\n", cDim, cReset)
	for _, ip := range ips {
		fmt.Printf("  %s%s%s\n", cYellow, ip, cReset)
	}
	fmt.Println()
	fmt.Printf("%sNOTE%s :: Enter one of the above IPs into PhantomAP remote on Termux\n", cDim, cReset)
	fmt.Printf("%sNOTE%s :: If connection is refused, add a firewall exception:\n", cDim, cReset)
	fmt.Printf("%s         netsh advfirewall firewall add rule name=PhantomAP dir=in action=allow protocol=TCP localport=%s%s\n", cDim, port, cReset)
	fmt.Println()

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("%s[ERR]%s BIND FAILED :: port %s :: %v\n", cRed, cReset, port, err)
		fmt.Printf("%s[ERR]%s Port may already be in use or blocked by firewall\n", cRed, cReset)
		fmt.Println()
		fmt.Print("Press Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("%s[..] LISTENING%s\n", cDim, cReset)
	fmt.Println()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logf("%s[!!]%s Accept error :: %v\n", cRed, cReset, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		logf("%s[>>]%s %s\n", cCyan, cReset, conn.RemoteAddr())
		go handleConn(conn)
	}
}
