// =================================================================================
// Filename: util-common.go
// Function: common utility functions
// Copyright: TeamGRIT, 2020
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------------
type ByteSize float64

const (
	_        = iota // ignore first value by assigning to blank identifier
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
	// PB
	// EB
	// ZB
	// YB
)

// ---------------------------------------------------------------------------------
type System struct {
	Type     string `json:"type"`
	Hostname string `json:"hostname,omitempty"`
	TempDir  string `json:"temp_dir,omitempty"`
	WorkDir  string `json:"work_dir,omitempty"`
	OS       string `json:"os,omitempty"`
	Arch     string `json:"arch,omitempty"`
	NCPU     int    `json:"ncpu,omitempty"`
	Golang   string `json:"golang,omitempty"`
}

func (si *System) String() (str string) {
	str = fmt.Sprintf("[%s] Hostname: %s", si.Type, si.Hostname)
	str += fmt.Sprintf(",\n\tTempDir: %s, WorkDir: %s", si.TempDir, si.WorkDir)
	str += fmt.Sprintf(",\n\tOS: %s, Arch: %s, nCPU: %d, Golang: %s", si.OS, si.Arch, si.NCPU, si.Golang)
	return
}

func GetSystemInfo() (si *System) {
	si = &System{}
	si.Type = "system"
	si.Hostname, _ = os.Hostname()
	si.TempDir = os.TempDir()
	si.WorkDir, _ = os.Getwd()
	si.OS = runtime.GOOS
	si.Arch = runtime.GOARCH
	si.NCPU = runtime.NumCPU()
	si.Golang = runtime.Version()
	return
}

// ---------------------------------------------------------------------------------
func IsValidMediaFile(fname string) bool {
	info, err := os.Stat(fname)
	if err != nil {
		log.Println("not found:", fname)
		return false
	}
	if info.IsDir() {
		log.Println("directory:", fname)
		return false
	}
	if info.Size() < KB {
		log.Println("too small:", info.Size())
		return false
	}
	return true
}

// ---------------------------------------------------------------------------------
func IsClosed(ch <-chan string) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

func IsNotClosed(ch <-chan string) bool {
	return !IsClosed(ch)
}

func SafeClose(ch chan string) {
	for IsNotClosed(ch) {
		log.Println("golang chan closing ...")
		close(ch)
	}
}

// ---------------------------------------------------------------------------------
func RespondHTTPState(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s", http.StatusText(status))
}

func RespondError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func RespondWithError(w http.ResponseWriter, code int, str string) {
	http.Error(w, str, code)
}

// ---------------------------------------------------------------------------------
func ParseQueryByID(r *http.Request) (m string) {
	// log.Println("ParseQuery:", r.URL)
	query := r.URL.Query()
	m = query.Get("id")
	return
}

func ParseQueryByItem(r *http.Request, item string) (m string) {
	// log.Println("ParseQueryByItem:", r.URL)
	query := r.URL.Query()
	m = query.Get(item)
	return
}

func ParseQueryByItemTicketOpt(r *http.Request, item string) (m, p, t, o string) {
	// log.Println("ParseQueryByItemOpt:", r.URL)
	query := r.URL.Query()
	m = query.Get(item)
	t = query.Get("ticket")
	p = query.Get("session")
	o = query.Get("opt")
	return
}

func ParseQueryChannelState(r *http.Request) (ch, st string) {
	// log.Println("ParseQueryChannelState:", r.URL)
	query := r.URL.Query()
	ch = query.Get("channel")
	st = query.Get("state")
	return
}

func ParseQueryUserPassword(r *http.Request) (user, pass string) {
	// log.Println("ParseQueryUserPassword:", r.URL)
	query := r.URL.Query()
	user = query.Get("user")
	pass = query.Get("pass")
	return
}

// ---------------------------------------------------------------------------------
func GenConnectURL(scheme, host, port, endpoint, query string) (url string) {
	url = fmt.Sprintf("%s://%s:%s/%s?%s", scheme, host, port, endpoint, query)
	log.Println(url)
	return
}

// ---------------------------------------------------------------------------------
func PrintJSONStruct(d interface{}) {
	data, err := json.MarshalIndent(d, "", "   ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(data))
}

// ---------------------------------------------------------------------------------
func FormatItem(d interface{}, format string) (str string, err error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(d, "", "   ")
		if err != nil {
			log.Println(err)
			return str, err
		}
		str = string(data)
	default:
		str = fmt.Sprintf("%s", reflect.ValueOf(d))
	}
	return
}

func FormatFilterStruct(d interface{}, format string) (str string, err error) {
	switch format {
	case "json":
		data, _ := json.MarshalIndent(d, "", "   ")
		str = string(data)
	default:
		// it use the previous string
	}
	return
}

// ---------------------------------------------------------------------------------
func IsInt(s string) bool {
	l := len(s)
	if strings.HasPrefix(s, "-") {
		l = l - 1
		s = s[1:]
	}

	reg := fmt.Sprintf("\\d{%d}", l)

	rs, err := regexp.MatchString(reg, s)
	if err != nil {
		return false
	}

	return rs
}

// ---------------------------------------------------------------------------------
func PrintRecoverStack() {
	if r := recover(); r != nil {
		log.Println("Recovered from", r)
		debug.PrintStack()
	}
}

func StringRecoverStack() (str string) {
	if r := recover(); r != nil {
		str = string(debug.Stack())
	}
	return
}

// ---------------------------------------------------------------------------------
// [Get local IP address in Go](https://gosamples.dev/local-ip-address/)
// ---------------------------------------------------------------------------------
func GetLocalIPString() (ipstr string) {
	conn, err := net.Dial("udp", "8.8.8.8:80") // Google DNS server (8.8.8.8)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ipstr = localAddr.IP.String()
	return
}

// ---------------------------------------------------------------------------------
// https://ifconfig.co/ip
// ---------------------------------------------------------------------------------
func GetExternalIPString() (ipstr string) {
	resp, err := http.Get("https://ipecho.net/plain")
	if err != nil {
		log.Println(err)
		resp, err = http.Get("http://myexternalip.com/raw")
		if err != nil {
			log.Println(err)
			return
		}
	}
	defer resp.Body.Close()

	ipaddr, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	exip := net.ParseIP(string(ipaddr))
	if IsPublicIP(exip) {
		ipstr = string(ipaddr)
		// log.Println("external ip:", exip)
	}
	return
}

// ---------------------------------------------------------------------------------
func IsValidHostCIDR(cidr, addr string) (valid bool) {
	log.Println("i.IsValidHostCIDR:", cidr, addr)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Println(err, host, port)
		return
	}
	if host == "::1" || host == "127.0.0.1" {
		log.Println("allow localhost:", host)
		return true
	}
	// check if Hostname is CIDR or IP
	if strings.Contains(host, "/") {
		return IsContainIP(cidr, host)
	} else {
		return IsMatchIP(cidr, host)
	}
}

// ---------------------------------------------------------------------------------
func IsMatchIP(cidr, host string) bool {
	if cidr == host {
		return true
	} else {
		return false
	}
}

// ---------------------------------------------------------------------------------
func IsContainIP(cidr, ip string) bool {
	log.Println(cidr, ip)
	_, netcidr, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Println(err)
		return false
	}
	netip := net.ParseIP(ip)
	log.Println(netcidr, netip)

	if netcidr.Contains(netip) {
		return true
	} else {
		return false
	}
}

// ---------------------------------------------------------------------------------
func IsPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------------
func GetExternalIP() (ips string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			log.Println(err)
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	err = fmt.Errorf("not connected to the network?")
	return
}

// ---------------------------------------------------------------------------------
func GetHostAddr(port int) (addr string, err error) {
	ips, err := GetExternalIP()
	if err != nil {
		log.Println(err)
		return
	}
	addr = fmt.Sprintf("%s:%d", ips, port)
	return
}

// ---------------------------------------------------------------------------------
func RecoverPanic() {
	if r := recover(); r != nil {
		log.Println("recovered from panic")
	}
}

// ---------------------------------------------------------------------------------
func GeneratePrivateCert() (err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Println(err)
		return
	}

	SNLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	SN, err := rand.Int(rand.Reader, SNLimit)
	if err != nil {
		log.Println(err)
		return
	}

	template := x509.Certificate{
		SerialNumber: SN,
		Subject: pkix.Name{
			Organization: []string{"TeamGRIT"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	template.DNSNames = append(template.DNSNames, "localhost")
	template.EmailAddresses = append(template.EmailAddresses, "sikang@teamgrit.kr")

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		log.Println(err)
		return
	}

	certFile, err := os.Create("cert/cert.pem")
	if err != nil {
		log.Println(err)
		return
	}
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		log.Println(err)
		return
	}

	keyFile, err := os.OpenFile("cert/key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Println(err)
		return
	}
	defer keyFile.Close()

	err = pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func GenerateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"spider-server"},
	}
}

// ---------------------------------------------------------------------------------
func Profiles(cpuprofile, memprofile, mutexprofile string) (err error) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Println("Create(cpuprofile):", err)
			return err
		}
		pprof.StartCPUProfile(f)
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	if memprofile != "" {
		defer func() {
			f, err := os.Create(memprofile)
			if err != nil {
				log.Println("Create(memprofile):", err)
				return
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}()
	}

	if mutexprofile != "" {
		runtime.SetMutexProfileFraction(1)
		defer func() {
			f, err := os.Create(mutexprofile)
			if err != nil {
				log.Println("Create(mutexprofile):", err)
				return
			}
			pprof.Lookup("mutex").WriteTo(f, 0)
			f.Close()
		}()
	}
	return
}

// ---------------------------------------------------------------------------------
func CaseInsensitiveContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

//=================================================================================
