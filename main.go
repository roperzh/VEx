package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/groob/plist"
)

func main() {
	ds, err := NewBoltStore("vex.db")
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()
	flRootsPath := flag.String("ca", "", "")
	fServerCertPath := flag.String("server-cert", "", "")
	fServerPrivateKeyPath := flag.String("server-private-key", "", "")

	fApnsCertPath := flag.String("apns-cert", "", "")
	fApnsKeyPath := flag.String("apns-key", "", "")
	flag.Parse()

	if *fApnsCertPath == "" || *fApnsKeyPath == "" {
		log.Fatal("apns-cert and -apns-key are required")
	}
	apnsClient, err := NewClient(*fApnsCertPath, *fApnsKeyPath)
	if err != nil {
		log.Fatalf("creating APNs client: %s", err)
	}

	commander := Commander{
		ds:         ds,
		apnsClient: apnsClient,
	}

	fPort := flag.String("port", ":9001", "")

	caCert, err := os.ReadFile(*flRootsPath)
	if err != nil {
		log.Fatal(err)
	}

	svc := &Service{ds: ds, commander: commander}

	writeErr := func(w http.ResponseWriter, status int, message string) {
		w.WriteHeader(status)
		w.Write([]byte(fmt.Sprintf(`{error: %q}`, message)))
	}

	http.HandleFunc("POST /devices/{udid}/command", func(w http.ResponseWriter, req *http.Request) {
		deviceUDID := req.PathValue("udid")

		device, err := ds.GetDevice(req.Context(), deviceUDID)
		if err != nil {
			// TODO: better error handling
			writeErr(w, http.StatusBadRequest, "device not found")
			return
		}

		rawCmd, err := io.ReadAll(req.Body)
		if err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("error reading request body: %s", err))
			return
		}

		var cmd CommandWrapper
		if err := plist.Unmarshal(rawCmd, &cmd); err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("request body must be a valid command: %s", err))
			return
		}

		if cmd.CommandUUID == "" {
			writeErr(w, http.StatusBadRequest, "command contains empty CommandUUID")
			return
		}

		if err := commander.Enqueue(req.Context(), []*Device{device}, map[string][]byte{cmd.CommandUUID: rawCmd}); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("enqueuing command: %s", err))

		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("GET /devices", func(w http.ResponseWriter, req *http.Request) {
		devices, err := ds.ListDevicesRaw()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		if _, err := w.Write(devices); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s: %s", req.Method, req.URL, b)
		var mdmReq MDMRequest
		if err := plist.NewXMLDecoder(bytes.NewReader(b)).Decode(&mdmReq); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := req.Context()
		device := mdmReq.Enrollment
		switch mdmReq.MessageType {
		case "Authenticate":
			err = svc.Authenticate(ctx, device, mdmReq.Authenticate)
		case "SetBootstrapToken":
			err = svc.SetBootstrapToken(ctx, device, mdmReq.SetBootstrapToken)
		case "TokenUpdate":
			err = svc.TokenUpdate(ctx, device, mdmReq.TokenUpdate)
		case "CheckOut":
			err = svc.CheckOut(ctx, device)
		case "DeclarativeManagement":
			err = svc.DeclarativeManagement(ctx, device, mdmReq.DeclarativeManagement)
		default:
			var body []byte
			body, err = svc.CommandsHandler(ctx, device, mdmReq.CommandResult, b)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
			}
			w.Write(body)
		}

		if err != nil {
			log.Printf("ERR handling request: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	server := &http.Server{
		Addr: *fPort,
		TLSConfig: &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		},
	}

	log.Printf("serving on port %s", *fPort)
	if err := server.ListenAndServeTLS(*fServerCertPath, *fServerPrivateKeyPath); err != nil {
		log.Fatalf("starting server: %s", err)
	}
}
