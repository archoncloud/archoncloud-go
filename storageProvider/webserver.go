package storageProvider

import (
	"context"
	. "github.com/archoncloud/archoncloud-go/common"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"time"
)

const (
	sslCert = "cert.pem"
	sslKey = "key.pem"
)

func httpInfo(w http.ResponseWriter, r *http.Request, message string) {
	LogInfo.Printf("%s: %s\n", requestInfo(r), message)
	_, _ = fmt.Fprintf(w, "%s\n", message)
}

func httpDebug(w http.ResponseWriter, r *http.Request, message string) {
	LogDebug.Printf("%s: %s\n", requestInfo(r), message)
	_, _ = fmt.Fprintf(w, "%s\n", message)
}

func httpInfo2(w http.ResponseWriter, r *http.Request, message string) {
	LogInfo.Printf("%s\n", requestInfo(r))
	_, _ = fmt.Fprintf(w, "%s\n", message)
}

func httpErr(w http.ResponseWriter, r *http.Request, err error, httpStatus int) {
	LogError.Printf("%s: %v\n", requestInfo(r), err)
	http.Error(w, "Error: "+err.Error(), httpStatus)
}

func httpErr500(w http.ResponseWriter, r *http.Request, err error) {
	httpErr(w, r, err, http.StatusInternalServerError)
}

func httpBadRequest(w http.ResponseWriter, r *http.Request, err error) {
	httpErr(w, r, err, http.StatusBadRequest)
}

func requestInfo(r *http.Request) string {
	return fmt.Sprintf("%q from %q", path.Base(r.URL.Path), r.RemoteAddr)
}

// ApiRouter sets up API routing
func ApiRouter() *mux.Router {
	// Initialize API endpoints
	r := mux.NewRouter()
	r.HandleFunc("/", infoHandler).Methods("GET")
	r.HandleFunc(StatsEndpoint, statsHandler).Methods("GET")
	r.HandleFunc(ContainsEndpoint, containsHandler).Methods("GET")
	r.HandleFunc(RetrieveEndpoint, retrieveHandler).Methods("GET")
	r.HandleFunc(SpProfilesEndpoint, spProfilesHandler).Methods("GET")
	r.HandleFunc("/log", ChainHandlers(showLogHandler, BasicAuthHandler)).Methods("GET")
	r.PathPrefix(UploadEndpoint).HandlerFunc(uploadHandler).Methods("POST")
	r.PathPrefix(DownloadEndpoint).HandlerFunc(downloadHandler).Methods("GET")
	//r.PathPrefix(HashEndpoint).Handler(
	//	http.StripPrefix(HashEndpoint, http.FileServer(hashesDir(HashesFolder)))).Methods("GET")
	//r.PathPrefix(ArcEndpoint).Handler(
	//	http.StripPrefix(ArcEndpoint, http.FileServer(shardsDir(ShardsFolder)))).Methods("GET")

	// This runs continuously
	go VerifyPendingUploads()
	return r
}

var rootFileServer = http.FileServer(http.Dir(StorageRoot))

func browseHandler(w http.ResponseWriter, r *http.Request) {
	rootFileServer.ServeHTTP(w, r)
}

func ApiPorts(conf *Configuration) (ports []int) {
	p, _ := strconv.Atoi(conf.Port)
	ports = append(ports, p)
	if FileExists(DefaultToExecutable(sslCert)) && FileExists(DefaultToExecutable(sslKey)) {
		ports = append(ports, p+1)
	}
	return
}

// RunWebServer starts the web servers
func RunWebServer() (err error) {
	conf := GetSPConfiguration()
	router := ApiRouter()
	ports := ApiPorts(conf)

	errsChannel := make(chan error, 3)
	var server, sslServer, browseServer *http.Server

	// For Ctrl+C
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, os.Kill)

	// Need to have a large timeout for large files
	//const timeout = 25*time.Minute
	// Starting HTTP server
	LogInfo.Printf(fmt.Sprintf("API on HTTP port %d\n", ports[0]))
	go func() {
		handler := cors.Default().Handler(router)
		server = &http.Server{
			Addr:           ":" + strconv.Itoa(ports[0]),
			Handler:        handler,
			//ReadTimeout:    timeout,
			//WriteTimeout:   timeout,
			MaxHeaderBytes: 1 << 20,
		}
		if err := server.ListenAndServe(); err != nil {
			errsChannel <- err
		}
	}()
	PortsUsed = append(PortsUsed, ports[0])

	if len(ports) > 1 {
		// We have certificate fiel
		// Starting HTTPS server
		LogInfo.Printf(fmt.Sprintf("API on HTTPS port %d\n", ports[1]))
		go func() {
			handler := cors.Default().Handler(router)
			sslServer = &http.Server{
				Addr:           ":" + strconv.Itoa(ports[1]),
				Handler:        handler,
				//ReadTimeout:    timeout,
				//WriteTimeout:   timeout,
				MaxHeaderBytes: 1 << 20,
			}
			// For now these files are in the same folder as the executable
			if err := sslServer.ListenAndServeTLS(sslCert, sslKey); err != nil {
				errsChannel <- err
			}
		}()
		PortsUsed = append(PortsUsed, ports[1])
	}

	// Starting browse server
	browsePortNumber := ports[0]+2
	LogInfo.Printf(fmt.Sprintf("Browser on HTTP port %d\n", browsePortNumber))
	PortsUsed = append(PortsUsed, browsePortNumber)
	go func() {
		http.Handle("/", ChainHandlers(browseHandler, BasicAuthHandler))
		browseServer = &http.Server{Addr: ":" + strconv.Itoa(browsePortNumber), Handler: nil}
		if err := browseServer.ListenAndServe(); err != nil {
			errsChannel <- err
		}
	}()

	sort.Ints(PortsUsed)
	LogInfo.Printf(fmt.Sprintf("Ports used: %v\n", PortsUsed))

	select {
	case err = <-errsChannel:
	case _ = <-signalChannel:
	}
	fmt.Println("\nStopping...")
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	if server != nil { server.Shutdown(ctx)}
	if sslServer != nil { sslServer.Shutdown(ctx) }
	if browseServer != nil { browseServer.Shutdown(ctx) }
	close(uploadsPendingChan)
	LogInfo.Printf("Stopped\n")
	return
}
