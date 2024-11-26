package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	// silakan baca keterangan pada jakpus [ malas ketik ulang. ]
	serverAddr  = "localhost:8080"
	peerAddr    = "localhost:8081"
	certFile    = "../certs/secret-rsjakut.crt"
	keyFile     = "../certs/secret-rsjakut.key"
	peerCAFile  = "../certs/secret-pubrsjakpus.crt"
	watchFolder = "data"
)

var conn *websocket.Conn // Declare conn untuk websocket connection

func main() {
	// Handle HTTP/2 dan validasi TLS dengan Websocket upgrade
	http.HandleFunc("/", hstsHandler)
	http.HandleFunc("/ws", handleWebSocket)

	server := &http.Server{
		Addr:      serverAddr,
		TLSConfig: configureTLS(certFile, keyFile, peerCAFile),
	}

	go monitorFolder(watchFolder)
	log.Println("Starting RS JAKUT Server on", serverAddr)
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}

// HSTS Handler
func hstsHandler(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || r.TLS.Version != tls.VersionTLS13 {
		http.Error(w, "Access Forbidden: Required TLS 1.3", http.StatusForbidden)
		return
	}
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains") // set header transport [HSTS]
	fmt.Fprintf(w, "RS DAVID JAKUT HTTP/2 Server with HSTS enabled")
}

// Function untuk konfigurasi tls
func configureTLS(certFile, keyFile, caFile string) *tls.Config {
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalf("Error loading CA certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Error loading server key pair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}
}

func monitorFolder(folderPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error initializing watcher:", err)
	}
	defer watcher.Close()

	err = watcher.Add(folderPath)
	if err != nil {
		log.Fatal("Error adding folder to watcher:", err)
	}

	// Wait for the connection to RS JAKPUS
	connectToPeer()

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Detected new file:", event.Name)
				sendFile(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("Error watching folder:", err)
		}
	}
}

// Connect to peer server (RS JAKPUS)
func connectToPeer() {
	for {
		caCert, err := ioutil.ReadFile(peerCAFile)
		if err != nil {
			log.Fatalf("Error loading CA certificate: %v", err)
		}

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCert)

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatalf("Error loading key pair: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            certPool,
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true, // tambah insecure soalnya pake self signed ssl
		}

		dialer := websocket.Dialer{TLSClientConfig: tlsConfig}
		conn, _, err = dialer.Dial("wss://"+peerAddr+"/ws", nil)
		if err != nil {
			log.Println("Waiting for RS JAKPUS connection...")
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Connected to RS JAKPUS")
		return
	}
}

// Send file data to RS JAKPUS
func sendFile(filePath string) {
	if conn == nil {
		log.Println("WebSocket connection is not established yet.")
		return
	}

	err := conn.WriteMessage(websocket.TextMessage, []byte(filePath))
	if err != nil {
		log.Println("Error sending file name:", err)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	stat, _ := file.Stat()
	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}

	err = conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		log.Println("Error sending file data:", err)
		return
	}
	log.Println("File sent successfully:", filePath)
}

// WebSocket handler for receiving file data from RS JAKPUS
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || r.TLS.Version != tls.VersionTLS13 {
		http.Error(w, "Access Forbidden: Invalid protocol", http.StatusForbidden)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}
	defer conn.Close()

	log.Println("Websocket connection established with RS JAKPUS")
	for {
		_, fileNameBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error receiving file name:", err)
			break
		}

		// Konvert Filename ke string
		fileName := string(fileNameBytes)

		_, fileData, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error receiving file data:", err)
			break
		}

		saveFileToData(fileName, fileData)
	}
}

func saveFileToData(fileName string, data []byte) {
	// filePath := fmt.Sprintf("%s", fileName)
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		log.Println("Error writing file:", err)
		return
	}

	log.Printf("File %s saved successfully in data folder", fileName)
}
