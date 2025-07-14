// =================================================================================
// Filename: main.go
// Function: Main function of moth streaming server
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2025
// =================================================================================
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"

	"github.com/rs/cors"
)

// ---------------------------------------------------------------------------------
func init() {
	log.SetPrefix("\033[32m[Moth]\033[0m ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// ---------------------------------------------------------------------------------
func initLogger(path string) (err error) {
	// if not defined, the file logger will be not used
	if path == "" {
		os.Remove("moth-current.log")
		return
	}
	os.MkdirAll(path, os.ModePerm) // make the log directory
	logger, err := rotatelogs.New(
		// fmt.Sprintf("%s/moth-%s.log", path, "%Y-%m-%d.%H:%M:%S"),
		fmt.Sprintf("%s/moth-%s.log", path, "%Y-%m-%d"),
		// rotatelogs.WithLinkName("moth-current.log"),
		rotatelogs.WithMaxAge(30*24*time.Hour),    // 30 days
		rotatelogs.WithRotationTime(24*time.Hour), // 24 hours = 1 day
	)
	if err != nil {
		log.Fatalln("logger setting error:", err)
	}
	log.SetOutput(logger)
	return
}

// ---------------------------------------------------------------------------------
// one studio per server, default 30 days license
var mConfig = NewConfigPointer()
var pStudio = NewStudioPointerWithName("Moth's Studio", 30)
var cWatcher = ConnectionWatcher{}

// ---------------------------------------------------------------------------------
// main server program of moth
func main() {
	// Read the configuration both from file and command options
	mConfig.readConfig("config", "conf/moth.json")
	// mConfig.writeConfig("config", "conf/moth.json")
	mConfig.procFlags()

	// run program modules by the run type
	mConfig.ProgramBase = fmt.Sprintf("%s/%s, %s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	switch mConfig.TypeRun {
	case "server":
		initLogger(mConfig.DirLog)
		// handler := mConfig.setupServerAPIs()
		handler := mConfig.setupMuxServerAPIs()

		mConfig.ProgramTitle = fmt.Sprintf("Moth Lite IoRT Media Server, v%s, (c)2025 TeamGRIT, Inc.", Version)
		fmt.Println(mConfig.ProgramTitle)
		log.Println(mConfig.ProgramTitle, "started on", mConfig.ProgramBase)
		if mConfig.PortPlain == 0 && mConfig.PortSecure == 0 {
			log.Println("no service ports:", mConfig.PortPlain, mConfig.PortSecure)
			os.Exit(1)
		}

		go RunPlainHTTPServer(pStudio, mConfig.PortPlain, handler)   // tcp, ws
		go RunSecureHTTPServer(pStudio, mConfig.PortSecure, handler) // tcp, wss
		go RunPlainTCPServer(pStudio, mConfig.TCPPlain)              // tcp
		go RunSecureTCPServer(pStudio, mConfig.TCPSecure)            // tcp

	// belows are clients for monitoring and testing
	case "manager":
		mConfig.ProgramTitle = fmt.Sprintf("Moth IoRT Media Manager (http), v%s, (c)2025 TeamGRIT, Inc.", Version)
		fmt.Println(mConfig.ProgramTitle)
		// url := fmt.Sprintf("https://%s:%d/manager/http/cmd", mConfig.HostAddr, mConfig.PortSecure)
		url := fmt.Sprintf("http://%s:%d/manager/http/cmd", mConfig.HostAddr, mConfig.PortPlain)
		StartHTTPManagerClient(url, mConfig.KeyManager)
		return
	case "manager2":
		mConfig.ProgramTitle = fmt.Sprintf("Moth IoRT Media Manager (ws), v%s, (c)2025 TeamGRIT, Inc.", Version)
		fmt.Println(mConfig.ProgramTitle)
		// url := fmt.Sprintf("wss://%s:%d/manager/ws/cmd", mConfig.HostAddr, mConfig.PortSecure)
		url := fmt.Sprintf("ws://%s:%d/manager/ws/cmd", mConfig.HostAddr, mConfig.PortPlain)
		StartWSManagerClient(url, mConfig.KeyManager)
		return
	case "monitor2":
		mConfig.ProgramTitle = fmt.Sprintf("Moth IoRT Media Monitor (ws), v%s, (c)2025 TeamGRIT, Inc.", Version)
		fmt.Println(mConfig.ProgramTitle)
		url := fmt.Sprintf("wss://%s:%d/monitor/ws/cmd", mConfig.HostAddr, mConfig.PortSecure)
		StartWSMonitorClient(url)
		return
	case "control":
		mConfig.ProgramTitle = fmt.Sprintf("Moth IoRT Media Controller (ws), v%s, (c)2022-5 TeamGRIT, Inc.", Version)
		fmt.Println(mConfig.ProgramTitle)
		url := fmt.Sprintf("wss://%s:%d/pang/ws/ctl?channel=%s", mConfig.HostAddr, mConfig.PortSecure, "c40hp6epjh65aeq6ne50")
		StartWSControlClient(url, "c40hp6epjh65aeq6ne50")
		return
	default:
		log.Println("not support run type:", mConfig.TypeRun)
		return
	}

	// Mandatory: read channel configuration
	err := pStudio.readObjectFileInArray("channel", "conf/channels.json")
	if err != nil {
		log.Println(err)
		return
	}
	// pStudio.writeObjectFileInArray("channel", "conf/channels.json") // for checking

	// Optional: read bridge configuration
	err = pStudio.readObjectFileInArray("bridge", "conf/bridges.json")
	if err != nil {
		log.Println(err)
		// return
	}
	// pStudio.writeObjectFileInArray("bridge", "conf/bridges.json") // for checking

	go StudioEventBroker(pStudio)
	RunSelfChecker(pStudio)
}

// ---------------------------------------------------------------------------------
// process command flag options
func (d *MothConfig) procFlags() {
	// temporal variable to handle input parameters
	mf := &MothConfig{}

	flag.StringVar(&mf.TypeRun, "rtype", d.TypeRun, "run type for server & tools: manager, ...")
	flag.IntVar(&mf.PortPlain, "portp", d.PortPlain, "http plain port")
	flag.IntVar(&mf.PortSecure, "ports", d.PortSecure, "http secure port, usually portp + 1")
	flag.StringVar(&mf.HostAddr, "haddr", d.HostAddr, "host address to connect")
	flag.StringVar(&mf.ServerURL, "surl", d.ServerURL, "server url to connect")
	flag.StringVar(&mf.KeyManager, "key", d.KeyManager, "server admin key")
	flag.Parse()

	if mf.TypeRun != "" {
		d.TypeRun = mf.TypeRun
	}
	if mf.PortPlain > 0 {
		d.PortPlain = mf.PortPlain
	}
	if mf.PortSecure > 0 {
		d.PortSecure = mf.PortSecure
	}
	if mf.HostAddr != "" {
		d.HostAddr = mf.HostAddr
	}
	if mf.ServerURL != "" {
		d.ServerURL = mf.ServerURL
	}
	if mf.KeyManager != "" {
		d.KeyManager = mf.KeyManager
	}

	d.ProgramTitle = fmt.Sprintf("Moth IoRT Media Server, v%s (%s, %s)",
		Version, d.Name, d.TypeRun)
}

// ---------------------------------------------------------------------------------
// server Mux API endpoints setting
func (d *MothConfig) setupMuxServerAPIs() (handler http.Handler) {
	mux := http.NewServeMux()

	// File server for test web pages
	wfs := http.FileServer(http.Dir("html"))
	mux.Handle("/static/", http.StripPrefix("/static/", wfs))

	mfs := http.FileServer(http.Dir("data"))
	mux.Handle("/data/", http.StripPrefix("/data/", mfs))

	mux.HandleFunc("/favicon.ico", FaviconHandler)

	// Control & Monitor APIs
	mux.HandleFunc("/monitor/", MonitorAPIHandler)
	mux.HandleFunc("/manager/", ManagerAPIHandler)

	// Live style APIs
	mux.HandleFunc("/pang/http/", PangHTTPHandler)
	mux.HandleFunc("/pang/ws/", PangWSHandler) // WebSocket(ws)
	mux.HandleFunc("/cast/ws/", CastWSHandler)

	// Signalling server APIs
	mux.HandleFunc("/signal/ws/", SignalWSHandler)

	if d.CORSAllow {
		handler = cors.AllowAll().Handler(mux)
	} else {
		handler = cors.Default().Handler(mux)
	}
	return
}

// ---------------------------------------------------------------------------------
// HTTP API endpoint handlers
// ---------------------------------------------------------------------------------
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	//defer log.Println("OUT FaviconHandler:", r.Method)
	defer r.Body.Close()
	http.ServeFile(w, r, "./html/favicon.ico")
}

// ---------------------------------------------------------------------------------
func MonitorAPIHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN MonitorAPIHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT MonitorAPIHandler:", r.Method)

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	switch r.URL.Path {
	case "/monitor/http/cmd":
		ProcMonitorHTTPCommand(w, r)
	case "/monitor/ws/cmd":
		ProcMonitorWSCommand(w, r)
	case "/monitor/ws/evt":
		ProcMonitorWSEvent(w, r)
	default:
		err = fmt.Errorf("unknown monitor api: %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func ManagerAPIHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN ManagerAPIHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT ManagerAPIHandler:", r.Method)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	switch r.URL.Path {
	case "/manager/http/cmd":
		ProcManagerHttpCommand(w, r)
	case "/manager/http/cmd2":
		ProcManagerHttpCommand2(w, r)
	case "/manager/ws/cmd":
		ProcManagerWsCommand(w, r)
	case "/manager/ws/evt":
		ProcManagerWsEvent(w, r)
	default:
		err = fmt.Errorf("unknown manager api: %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func CastWSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN CastWSHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT CastWSHandler:", r.URL)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	qo, err := GetQueryOptionFromRequest(r)
	if err != nil {
		log.Println(err)
		return
	}

	ws, err := UpgradeToWebSocket(w, r, 2048)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	switch r.URL.Path {
	case "/cast/ws/pub":
		err = CastWSPublisher(ws, qo)
	case "/cast/ws/sub":
		err = CastWSSubscriber(ws, qo)
	default:
		err = fmt.Errorf("unknown api %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func PangHTTPHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN PangHTTPHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT PangHTTPHandler:", r.URL)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	qo, err := GetQueryOptionFromRequest(r)
	if err != nil {
		log.Println(err)
		return
	}

	switch r.URL.Path {
	case "/pang/http/pub":
		err = PangLivePublisher(w, r, qo)
	case "/pang/http/sub":
		err = PangLiveSubscriber(w, r, qo)
	default:
		err = fmt.Errorf("unknown api %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func PangWSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN PangWSHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT PangWSHandler:", r.URL)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	qo, err := GetQueryOptionFromRequest(r)
	if err != nil {
		log.Println(err)
		return
	}

	ws, err := UpgradeToWebSocket(w, r, 2048)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	switch r.URL.Path {
	case "/pang/ws/eco", "/pang/ws/echo":
		err = PangWSReflector(ws, qo)
	case "/pang/ws/ctl":
		err = PangWSController(ws, qo)
	case "/pang/ws/pub":
		err = PangWSPublisher(ws, qo)
	case "/pang/ws/sub":
		err = PangWSSubscriber(ws, qo)
	case "/pang/ws/meb": // medusa, broadcast mode
		err = PangWSMedusa(ws, qo)
	case "/pang/ws/evt":
		err = PangWSEventer(ws, qo)
	default:
		err = fmt.Errorf("unknown api %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func PangUDPHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN PangUDPHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT PangUDPHandler:", r.URL)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	qo, err := GetQueryOptionFromRequest(r)
	if err != nil {
		log.Println(err)
		return
	}

	ws, err := UpgradeToWebSocket(w, r, 2048)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	switch r.URL.Path {
	case "/pang/udp/pub":
		err = PangUDPPublisher(ws, qo)
	case "/pang/udp/sub":
		err = PangUDPSubscriber(ws, qo)
	default:
		err = fmt.Errorf("unknown api %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
func SignalWSHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("IN SignalWSHandler:", r.Method, r.URL, r.RemoteAddr)
	defer log.Println("OUT SignalWSHandler:", r.URL)
	defer r.Body.Close()

	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			return
		}
	}()

	qo, err := GetQueryOptionFromRequest(r)
	if err != nil {
		log.Println(err)
		return
	}

	ws, err := UpgradeToWebSocket(w, r, 2048)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	switch r.URL.Path {
	case "/signal/ws/moth":
		err = SignalWSServer(ws, qo)
	default:
		err = fmt.Errorf("unknown api %s", r.URL.Path)
	}
}

// ---------------------------------------------------------------------------------
// common base function servers for API endpoints, they should be never stopped
// ---------------------------------------------------------------------------------
// Plain API server supporting http and ws
func RunPlainHTTPServer(pst *Studio, port int, handler http.Handler) {
	if port == 0 {
		log.Println("invalid plain http port:", port)
		return
	}

	w := pStudio.addNewWorkerWithParams("/server/http/api", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	w.Addr = fmt.Sprintf(":%d", port)
	w.Proto = "http/tcp"
	log.Println("plain http(tcp) server started on", w.Addr)

	server := &http.Server{
		ConnState: cWatcher.OnStateChange,
		Addr:      w.Addr,
		Handler:   handler,
		// ReadTimeout: time.Second * 30,
		// WriteTimeout: time.Second * 30, // CAUTION: Don't this if SSE is used
		IdleTimeout: time.Second * 30,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// Secure API server supporting https and wss (tls version)
func RunSecureHTTPServer(pst *Studio, port int, handler http.Handler) {
	if port == 0 {
		log.Println("invalid secure http port:", port)
		return
	}

	w := pStudio.addNewWorkerWithParams("/server/https/api", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	w.Addr = fmt.Sprintf(":%d", port)
	w.Proto = "https/tcp"
	log.Println("secure http(tcp) server started on", w.Addr)

	server := &http.Server{
		ConnState: cWatcher.OnStateChange,
		Addr:      w.Addr,
		Handler:   handler,
		// ReadTimeout:  time.Second * 30,
		// WriteTimeout: time.Second * 30, // CAUTION: Don't this if SSE is used
		IdleTimeout: time.Second * 30,
		// TLSConfig:    tlsConfig,
	}
	err := server.ListenAndServeTLS(mConfig.PEMPublic, mConfig.PEMPrivate)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------------
// Status checker for internal server (studio)
func RunSelfChecker(pst *Studio) (err error) {
	defer func() {
		err = fmt.Errorf("server license is expired! so, going down")
		log.Println(err)
		os.Exit(0)
	}()

	err = mConfig.license.parseKeyString(mConfig.KeyLicense)
	if err != nil {
		log.Println(err)
		mConfig.KeyLicense = "Demo License (please register!)"
	}

	w := pStudio.addNewWorkerWithParams("/checker/moth/data", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for mConfig.isValid() {
		select {
		case sig := <-c:
			log.Println("Shutting down by", sig)
			os.Exit(0)
		case <-ticker.C:
			pst.AtUsed = time.Now()
			pst.cleanChannels()
			pst.cleanPunches()
			pst.checkBridges("ever")
		}
		w.AtUsed = time.Now()
	}
	return
}

//=================================================================================
